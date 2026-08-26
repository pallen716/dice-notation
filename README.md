# dice-notation

A Go library for parsing and rolling tabletop dice notation: `3d6+2`,
`4d6kh3` (roll four six-sided dice, keep the highest three), `d%`, and so
on. Every dice bot, character sheet tool, or combat tracker ends up needing
this exact parser, and it's fiddly enough (keep/drop semantics, ties,
modifiers) that it's worth getting right once instead of copy-pasting a
regex around.

This is a library, not a command-line tool. It's meant to be embedded in
whatever you're building — a Discord bot, a web API, a terminal app — which
is also why every result knows how to render itself as either plain text or
JSON: bots want the former for the channel and the latter for logging or
for a companion API.

## Install

```
go get github.com/pallen716/dice-notation
```

## Usage

```go
package main

import (
	"fmt"
	"log"

	"github.com/pallen716/dice-notation"
)

func main() {
	result, err := dice.Roll("4d6kh3+2")
	if err != nil {
		log.Fatal(err)
	}

	// Human readable, e.g.: "4d6kh3+2: [2 5 6 1] (dropped [1]) +2 = 15"
	fmt.Println(result)

	// Same result, JSON mode, for a --json flag or a log line:
	out, err := dice.Format(result, true)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
}
```

`dice.Roll` is a convenience wrapper around `Parse` + `Expression.Roll` for
one-off rolls. If you're rolling the same expression repeatedly, or you
need reproducible output for a test, parse once and supply your own random
source:

```go
expr, err := dice.Parse("2d20kl1") // disadvantage
if err != nil {
	log.Fatal(err)
}

rng := rand.New(rand.NewSource(42))
result, err := expr.Roll(rng)
```

## Notation supported so far

- `NdS` — roll N dice with S sides (`N` defaults to 1: `d20` == `1d20`)
- `d%` — shorthand for a hundred-sided die
- `khN` / `klN` — keep the highest/lowest N dice
- `dhN` / `dlN` — drop the highest/lowest N dice
- a trailing `+N` or `-N` modifier

Keep/drop and the modifier are each optional and each limited to one per
expression for now — see the roadmap below for what's not here yet.

## Output

`Result` carries every roll, which ones were kept vs. dropped, the
modifier, and the total:

```go
type Result struct {
	Expression string
	Rolls      []int
	Kept       []int
	Dropped    []int
	Modifier   int
	Total      int
}
```

`Result.String()` gives you the human-readable line shown above.
`Result.JSON()` gives you the same data as indented JSON. `Format(result,
asJSON)` picks between the two, which is the shape you want if you're
wiring this up behind an actual `--json` flag in your own tool.

## License

MIT, see [LICENSE](LICENSE).
