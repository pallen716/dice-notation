// Package dice parses and rolls tabletop dice notation such as "3d6+2" or
// "4d6kh3" (roll four six-sided dice, keep the highest three). Expressions
// can chain multiple dice groups and flat modifiers together, as in
// "2d6+1d4-1". Dice groups can also explode, as in "3d6!", meaning any die
// that rolls its maximum value is rolled again with the extra result added
// to the pool.
package dice

import (
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"time"
)

// termDicePattern matches a single dice group with no leading sign: an
// optional count, "d", sides (or "%"), an optional explode marker ("!"),
// and an optional keep/drop modifier (kh/kl/dh/dl followed by a count).
// Examples: "d20", "3d6", "4d6kh3", "3d6!", "3d6!kh2".
var termDicePattern = regexp.MustCompile(`^(\d*)d(\d+|%)(!)?((?:kh|kl|dh|dl)\d+)?$`)

// termConstPattern matches a flat numeric term with no leading sign, e.g.
// the "2" in "3d6+2".
var termConstPattern = regexp.MustCompile(`^(\d+)$`)

const (
	maxDice  = 1000
	maxSides = 100000

	// maxExplosions caps how many extra dice a single die can chain into.
	// explode is rejected at parse time for sides < 2, so in practice a
	// chain this long would take a run of luck astronomically unlikely to
	// occur; the cap exists only to bound worst-case work.
	maxExplosions = 100
)

// Term is one piece of an Expression: either a dice group (Sides > 0) or a
// flat constant (Sides == 0), combined into the total according to Sign.
type Term struct {
	Sign        int
	Count       int
	Sides       int
	Explode     bool
	KeepHighest int
	KeepLowest  int
	DropHighest int
	DropLowest  int
	Constant    int
}

// Expression is a parsed dice notation string, ready to roll as many times
// as needed without re-parsing.
type Expression struct {
	Raw   string
	Terms []Term
}

// Parse turns a notation string into an Expression. The notation is one or
// more terms joined by "+" or "-". Each term is either a flat integer or a
// dice group in NdS form, optionally followed by an explode marker ("!")
// and then one keep/drop clause (kh, kl, dh, dl, each followed by a
// count). "d%" is shorthand for a hundred-sided die. At least one term
// must be a dice group; a bare number like "5" is rejected.
func Parse(notation string) (*Expression, error) {
	if notation == "" {
		return nil, fmt.Errorf("dice: empty notation")
	}

	tokens := splitTerms(notation)
	terms := make([]Term, 0, len(tokens))
	hasDice := false
	for _, tok := range tokens {
		term, err := parseTerm(tok)
		if err != nil {
			return nil, fmt.Errorf("dice: invalid notation %q: %w", notation, err)
		}
		if term.Sides > 0 {
			hasDice = true
		}
		terms = append(terms, term)
	}
	if !hasDice {
		return nil, fmt.Errorf("dice: notation %q has no dice terms", notation)
	}

	return &Expression{Raw: notation, Terms: terms}, nil
}

// splitTerms breaks a notation string into signed term substrings, e.g.
// "4d6kh3+1d4-2" becomes ["4d6kh3", "+1d4", "-2"]. The first term keeps its
// implicit sign (no leading "+" or "-" required). Splitting on raw "+"/"-"
// bytes is safe because no valid term contains either character internally.
func splitTerms(notation string) []string {
	var terms []string
	start := 0
	for i := 1; i < len(notation); i++ {
		if notation[i] == '+' || notation[i] == '-' {
			terms = append(terms, notation[start:i])
			start = i
		}
	}
	return append(terms, notation[start:])
}

// parseTerm parses a single signed term produced by splitTerms.
func parseTerm(token string) (Term, error) {
	sign := 1
	body := token
	switch token[0] {
	case '+':
		body = token[1:]
	case '-':
		sign = -1
		body = token[1:]
	}
	if body == "" {
		return Term{}, fmt.Errorf("empty term %q", token)
	}

	if m := termConstPattern.FindStringSubmatch(body); m != nil {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return Term{}, fmt.Errorf("invalid constant %q: %w", body, err)
		}
		return Term{Sign: sign, Constant: n}, nil
	}

	m := termDicePattern.FindStringSubmatch(body)
	if m == nil {
		return Term{}, fmt.Errorf("invalid term %q", token)
	}

	count := 1
	if m[1] != "" {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return Term{}, fmt.Errorf("invalid count in %q: %w", token, err)
		}
		count = n
	}
	if count < 1 || count > maxDice {
		return Term{}, fmt.Errorf("count %d out of range (1-%d)", count, maxDice)
	}

	var sides int
	if m[2] == "%" {
		sides = 100
	} else {
		n, err := strconv.Atoi(m[2])
		if err != nil {
			return Term{}, fmt.Errorf("invalid sides in %q: %w", token, err)
		}
		sides = n
	}
	if sides < 1 || sides > maxSides {
		return Term{}, fmt.Errorf("sides %d out of range (1-%d)", sides, maxSides)
	}

	term := Term{Sign: sign, Count: count, Sides: sides}

	if m[3] == "!" {
		if sides < 2 {
			return Term{}, fmt.Errorf("sides %d too small to explode in %q", sides, token)
		}
		term.Explode = true
	}

	if m[4] != "" {
		kind := m[4][:2]
		n, err := strconv.Atoi(m[4][2:])
		if err != nil {
			return Term{}, fmt.Errorf("invalid keep/drop count in %q: %w", token, err)
		}
		if n < 1 || n > count {
			return Term{}, fmt.Errorf("keep/drop count %d exceeds dice count %d", n, count)
		}
		switch kind {
		case "kh":
			term.KeepHighest = n
		case "kl":
			term.KeepLowest = n
		case "dh":
			term.DropHighest = n
		case "dl":
			term.DropLowest = n
		}
	}

	return term, nil
}

// TermResult is the outcome of rolling (or evaluating) a single Term.
type TermResult struct {
	Sign     int   `json:"sign"`
	Count    int   `json:"count,omitempty"`
	Sides    int   `json:"sides,omitempty"`
	Rolls    []int `json:"rolls,omitempty"`
	Kept     []int `json:"kept,omitempty"`
	Dropped  []int `json:"dropped,omitempty"`
	Constant int   `json:"constant,omitempty"`
	Subtotal int   `json:"subtotal"`
}

// Result is the outcome of rolling an Expression, broken down term by term
// in the same order the notation listed them.
type Result struct {
	Expression string       `json:"expression"`
	Terms      []TermResult `json:"terms"`
	Total      int          `json:"total"`
}

// Roll rolls the expression using the supplied random source. Taking the
// source as a parameter, rather than reaching for a package global, is what
// makes results reproducible in tests.
func (e *Expression) Roll(rng *rand.Rand) (*Result, error) {
	result := &Result{Expression: e.Raw, Terms: make([]TermResult, len(e.Terms))}

	total := 0
	for i, term := range e.Terms {
		tr := rollTerm(term, rng)
		result.Terms[i] = tr
		total += tr.Subtotal
	}
	result.Total = total

	return result, nil
}

// rollTerm rolls a single dice term, or evaluates a constant term, and
// applies its sign to produce a subtotal.
func rollTerm(term Term, rng *rand.Rand) TermResult {
	tr := TermResult{Sign: term.Sign}

	if term.Sides == 0 {
		tr.Constant = term.Constant
		tr.Subtotal = term.Sign * term.Constant
		return tr
	}

	rolls := rollDice(term.Count, term.Sides, term.Explode, func() int {
		return 1 + rng.Intn(term.Sides)
	})
	n := len(rolls)

	keepCount := n
	keepHighest := true
	switch {
	case term.KeepHighest > 0:
		keepCount, keepHighest = term.KeepHighest, true
	case term.KeepLowest > 0:
		keepCount, keepHighest = term.KeepLowest, false
	case term.DropHighest > 0:
		keepCount, keepHighest = n-term.DropHighest, false
	case term.DropLowest > 0:
		keepCount, keepHighest = n-term.DropLowest, true
	}

	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool { return rolls[idx[a]] < rolls[idx[b]] })

	keepSet := make(map[int]bool, keepCount)
	if keepHighest {
		for _, i := range idx[n-keepCount:] {
			keepSet[i] = true
		}
	} else {
		for _, i := range idx[:keepCount] {
			keepSet[i] = true
		}
	}

	kept := make([]int, 0, keepCount)
	dropped := make([]int, 0, n-keepCount)
	sum := 0
	for i, v := range rolls {
		if keepSet[i] {
			kept = append(kept, v)
			sum += v
		} else {
			dropped = append(dropped, v)
		}
	}

	tr.Count = term.Count
	tr.Sides = term.Sides
	tr.Rolls = rolls
	tr.Kept = kept
	tr.Dropped = dropped
	tr.Subtotal = term.Sign * sum
	return tr
}

// rollDice rolls count dice by calling next once per die. When explode is
// true, a die that comes up at its maximum value triggers an extra call to
// next, chained until a non-maximum value appears or maxExplosions is hit.
func rollDice(count, sides int, explode bool, next func() int) []int {
	rolls := make([]int, 0, count)
	for i := 0; i < count; i++ {
		v := next()
		rolls = append(rolls, v)
		for chain := 0; explode && v == sides && chain < maxExplosions; chain++ {
			v = next()
			rolls = append(rolls, v)
		}
	}
	return rolls
}

// Roll parses and rolls a notation string in one step, using a source seeded
// from the current time. Callers that need reproducible results should call
// Parse and Expression.Roll directly with their own *rand.Rand.
func Roll(notation string) (*Result, error) {
	expr, err := Parse(notation)
	if err != nil {
		return nil, err
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return expr.Roll(rng)
}
