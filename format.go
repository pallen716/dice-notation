package dice

import (
	"encoding/json"
	"fmt"
	"strings"
)

// String renders the result the way someone reading a terminal wants it:
// the notation, then each term's rolls (or value) in order, what got
// dropped (if anything), and the total.
func (r *Result) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: ", r.Expression)
	for i, t := range r.Terms {
		switch {
		case i == 0 && t.Sign < 0:
			b.WriteString("-")
		case i > 0 && t.Sign < 0:
			b.WriteString(" - ")
		case i > 0:
			b.WriteString(" + ")
		}
		if t.Sides > 0 {
			fmt.Fprintf(&b, "%v", t.Rolls)
			if len(t.Rerolled) > 0 {
				fmt.Fprintf(&b, " (rerolled %v)", t.Rerolled)
			}
			if len(t.Dropped) > 0 {
				fmt.Fprintf(&b, " (dropped %v)", t.Dropped)
			}
		} else {
			fmt.Fprintf(&b, "%d", t.Constant)
		}
	}
	fmt.Fprintf(&b, " = %d", r.Total)
	return b.String()
}

// JSON renders the result as indented JSON, for callers that hand the
// output to another program instead of a person.
func (r *Result) JSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Format picks between the two output modes. It exists so a thin CLI or
// bot command built on top of this package can wire a --json flag straight
// to asJSON without duplicating the branch everywhere it prints a result.
func Format(r *Result, asJSON bool) (string, error) {
	if asJSON {
		return r.JSON()
	}
	return r.String(), nil
}
