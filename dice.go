// Package dice parses and rolls tabletop dice notation such as "3d6+2" or
// "4d6kh3" (roll four six-sided dice, keep the highest three).
package dice

import (
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"time"
)

// notationPattern matches: optional count, "d", sides (or "%"), an optional
// keep/drop modifier (kh/kl/dh/dl followed by a count), and an optional
// trailing +/- modifier. Examples: "d20", "3d6", "4d6kh3", "2d20kl1+1".
var notationPattern = regexp.MustCompile(`^(\d*)d(\d+|%)((?:kh|kl|dh|dl)\d+)?([+-]\d+)?$`)

const (
	maxDice  = 1000
	maxSides = 100000
)

// Expression is a parsed dice notation string, ready to roll as many times
// as needed without re-parsing.
type Expression struct {
	Raw         string
	Count       int
	Sides       int
	KeepHighest int
	KeepLowest  int
	DropHighest int
	DropLowest  int
	Modifier    int
}

// Parse turns a notation string into an Expression. It accepts NdS core
// notation, one optional keep/drop clause (kh, kl, dh, dl, each followed by
// a count), and one optional trailing +N or -N modifier. "d%" is shorthand
// for a hundred-sided die.
func Parse(notation string) (*Expression, error) {
	m := notationPattern.FindStringSubmatch(notation)
	if m == nil {
		return nil, fmt.Errorf("dice: invalid notation %q", notation)
	}

	count := 1
	if m[1] != "" {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return nil, fmt.Errorf("dice: invalid count in %q: %w", notation, err)
		}
		count = n
	}
	if count < 1 || count > maxDice {
		return nil, fmt.Errorf("dice: count %d out of range (1-%d)", count, maxDice)
	}

	var sides int
	if m[2] == "%" {
		sides = 100
	} else {
		n, err := strconv.Atoi(m[2])
		if err != nil {
			return nil, fmt.Errorf("dice: invalid sides in %q: %w", notation, err)
		}
		sides = n
	}
	if sides < 1 || sides > maxSides {
		return nil, fmt.Errorf("dice: sides %d out of range (1-%d)", sides, maxSides)
	}

	expr := &Expression{Raw: notation, Count: count, Sides: sides}

	if m[3] != "" {
		kind := m[3][:2]
		n, err := strconv.Atoi(m[3][2:])
		if err != nil {
			return nil, fmt.Errorf("dice: invalid keep/drop count in %q: %w", notation, err)
		}
		if n < 1 || n > count {
			return nil, fmt.Errorf("dice: keep/drop count %d exceeds dice count %d", n, count)
		}
		switch kind {
		case "kh":
			expr.KeepHighest = n
		case "kl":
			expr.KeepLowest = n
		case "dh":
			expr.DropHighest = n
		case "dl":
			expr.DropLowest = n
		}
	}

	if m[4] != "" {
		n, err := strconv.Atoi(m[4])
		if err != nil {
			return nil, fmt.Errorf("dice: invalid modifier in %q: %w", notation, err)
		}
		expr.Modifier = n
	}

	return expr, nil
}

// Result is the outcome of rolling an Expression. Rolls holds every die in
// the order it was rolled; Kept and Dropped partition that same set of
// values according to the expression's keep/drop clause.
type Result struct {
	Expression string `json:"expression"`
	Rolls      []int  `json:"rolls"`
	Kept       []int  `json:"kept"`
	Dropped    []int  `json:"dropped,omitempty"`
	Modifier   int    `json:"modifier"`
	Total      int    `json:"total"`
}

// Roll rolls the expression using the supplied random source. Taking the
// source as a parameter, rather than reaching for a package global, is what
// makes results reproducible in tests.
func (e *Expression) Roll(rng *rand.Rand) (*Result, error) {
	rolls := make([]int, e.Count)
	for i := range rolls {
		rolls[i] = 1 + rng.Intn(e.Sides)
	}

	keepCount := e.Count
	keepHighest := true
	switch {
	case e.KeepHighest > 0:
		keepCount, keepHighest = e.KeepHighest, true
	case e.KeepLowest > 0:
		keepCount, keepHighest = e.KeepLowest, false
	case e.DropHighest > 0:
		keepCount, keepHighest = e.Count-e.DropHighest, false
	case e.DropLowest > 0:
		keepCount, keepHighest = e.Count-e.DropLowest, true
	}

	idx := make([]int, e.Count)
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool { return rolls[idx[a]] < rolls[idx[b]] })

	keepSet := make(map[int]bool, keepCount)
	if keepHighest {
		for _, i := range idx[e.Count-keepCount:] {
			keepSet[i] = true
		}
	} else {
		for _, i := range idx[:keepCount] {
			keepSet[i] = true
		}
	}

	kept := make([]int, 0, keepCount)
	dropped := make([]int, 0, e.Count-keepCount)
	for i, v := range rolls {
		if keepSet[i] {
			kept = append(kept, v)
		} else {
			dropped = append(dropped, v)
		}
	}

	total := e.Modifier
	for _, v := range kept {
		total += v
	}

	return &Result{
		Expression: e.Raw,
		Rolls:      rolls,
		Kept:       kept,
		Dropped:    dropped,
		Modifier:   e.Modifier,
		Total:      total,
	}, nil
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
