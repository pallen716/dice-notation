package dice

import (
	"math/rand"
	"testing"
)

// BenchmarkParseSimple covers the common case a bot command hits on every
// message: a short expression with no keep/drop, explode, or reroll clause.
func BenchmarkParseSimple(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Parse("3d6+2"); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseComplex exercises every optional clause the grammar
// supports, so the regexp match and follow-up parsing do the most work
// they can for a single term.
func BenchmarkParseComplex(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Parse("4d6!ro1kh3"); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseMultiTerm parses a notation with several dice groups and
// constants chained together, which is the shape a character sheet's
// "roll everything" button tends to produce.
func BenchmarkParseMultiTerm(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Parse("2d6+1d4+3d8kh2-1"); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseLargeBatch parses a single term at the count ceiling, to
// show that parse cost stays flat with dice count since the regexp only
// ever matches the count digits, not one match per die.
func BenchmarkParseLargeBatch(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Parse("1000d6"); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRollLargeBatch rolls at the count ceiling, where the sort and
// keep-set bookkeeping in rollTerm scale with dice count and dominate the
// per-call cost.
func BenchmarkRollLargeBatch(b *testing.B) {
	expr, err := Parse("1000d6kh500")
	if err != nil {
		b.Fatal(err)
	}
	rng := rand.New(rand.NewSource(1))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := expr.Roll(rng); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRollLargeBatchExploding adds exploding dice to the large batch,
// since each explosion re-enters rollDice's inner loop and can grow the
// roll slice past its initial capacity.
func BenchmarkRollLargeBatchExploding(b *testing.B) {
	expr, err := Parse("1000d6!")
	if err != nil {
		b.Fatal(err)
	}
	rng := rand.New(rand.NewSource(1))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := expr.Roll(rng); err != nil {
			b.Fatal(err)
		}
	}
}
