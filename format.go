package dice

import (
	"encoding/json"
	"fmt"
	"strings"
)

// String renders the result the way someone reading a terminal wants it:
// the notation, the individual rolls, what got dropped (if anything), and
// the total.
func (r *Result) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %v", r.Expression, r.Rolls)
	if len(r.Dropped) > 0 {
		fmt.Fprintf(&b, " (dropped %v)", r.Dropped)
	}
	if r.Modifier != 0 {
		fmt.Fprintf(&b, " %+d", r.Modifier)
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
