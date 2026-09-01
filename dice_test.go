package dice

import (
	"math/rand"
	"testing"
)

func TestParseBasic(t *testing.T) {
	expr, err := Parse("3d6+2")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(expr.Terms) != 2 {
		t.Fatalf("expected 2 terms, got %d", len(expr.Terms))
	}
	if expr.Terms[0].Count != 3 || expr.Terms[0].Sides != 6 {
		t.Fatalf("unexpected dice term: %+v", expr.Terms[0])
	}
	if expr.Terms[1].Sign != 1 || expr.Terms[1].Constant != 2 {
		t.Fatalf("unexpected constant term: %+v", expr.Terms[1])
	}
}

func TestParseKeepHighest(t *testing.T) {
	expr, err := Parse("4d6kh3")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if expr.Terms[0].KeepHighest != 3 {
		t.Fatalf("expected KeepHighest 3, got %d", expr.Terms[0].KeepHighest)
	}
}

func TestParsePercentile(t *testing.T) {
	expr, err := Parse("d%")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if expr.Terms[0].Count != 1 || expr.Terms[0].Sides != 100 {
		t.Fatalf("unexpected expression: %+v", expr.Terms[0])
	}
}

func TestParseInvalid(t *testing.T) {
	if _, err := Parse("not-dice"); err == nil {
		t.Fatal("expected an error for invalid notation")
	}
}

func TestParseNoDiceTerms(t *testing.T) {
	if _, err := Parse("5"); err == nil {
		t.Fatal("expected an error for a notation with no dice terms")
	}
}

func TestParseTrailingSign(t *testing.T) {
	if _, err := Parse("2d6+"); err == nil {
		t.Fatal("expected an error for a dangling + with no term")
	}
}

func TestParseKeepExceedsCount(t *testing.T) {
	if _, err := Parse("2d6kh3"); err == nil {
		t.Fatal("expected an error when keep count exceeds dice count")
	}
}

func TestParseMultiTerm(t *testing.T) {
	expr, err := Parse("2d6+1d4-1")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(expr.Terms) != 3 {
		t.Fatalf("expected 3 terms, got %d", len(expr.Terms))
	}
	if expr.Terms[0].Count != 2 || expr.Terms[0].Sides != 6 || expr.Terms[0].Sign != 1 {
		t.Fatalf("unexpected first term: %+v", expr.Terms[0])
	}
	if expr.Terms[1].Count != 1 || expr.Terms[1].Sides != 4 || expr.Terms[1].Sign != 1 {
		t.Fatalf("unexpected second term: %+v", expr.Terms[1])
	}
	if expr.Terms[2].Sign != -1 || expr.Terms[2].Constant != 1 {
		t.Fatalf("unexpected third term: %+v", expr.Terms[2])
	}
}

func TestRollDeterministic(t *testing.T) {
	expr, err := Parse("2d6+1")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	rng := rand.New(rand.NewSource(42))
	result, err := expr.Roll(rng)
	if err != nil {
		t.Fatalf("Roll returned error: %v", err)
	}
	if len(result.Terms) != 2 || len(result.Terms[0].Rolls) != 2 {
		t.Fatalf("expected 2 terms with 2 dice rolls, got %+v", result.Terms)
	}
	want := result.Terms[0].Rolls[0] + result.Terms[0].Rolls[1] + 1
	if result.Total != want {
		t.Fatalf("total = %d, want %d", result.Total, want)
	}
}

func TestRollKeepHighestDropsRest(t *testing.T) {
	expr, err := Parse("4d6kh3")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	rng := rand.New(rand.NewSource(7))
	result, err := expr.Roll(rng)
	if err != nil {
		t.Fatalf("Roll returned error: %v", err)
	}
	kept, dropped := result.Terms[0].Kept, result.Terms[0].Dropped
	if len(kept) != 3 || len(dropped) != 1 {
		t.Fatalf("expected 3 kept and 1 dropped, got kept=%v dropped=%v", kept, dropped)
	}
	sum := 0
	for _, v := range kept {
		sum += v
	}
	if sum != result.Total {
		t.Fatalf("total = %d, want sum of kept %d", result.Total, sum)
	}
}

func TestRollMultiTerm(t *testing.T) {
	expr, err := Parse("2d6+1d4-1")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	rng := rand.New(rand.NewSource(3))
	result, err := expr.Roll(rng)
	if err != nil {
		t.Fatalf("Roll returned error: %v", err)
	}
	if len(result.Terms) != 3 {
		t.Fatalf("expected 3 terms, got %d", len(result.Terms))
	}

	want := 0
	for _, tr := range result.Terms {
		want += tr.Subtotal
	}
	if result.Total != want {
		t.Fatalf("total = %d, want sum of subtotals %d", result.Total, want)
	}

	last := result.Terms[2]
	if last.Sign != -1 || last.Constant != 1 || last.Subtotal != -1 {
		t.Fatalf("unexpected constant term result: %+v", last)
	}
}

func TestFormatJSON(t *testing.T) {
	expr, err := Parse("1d20")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	rng := rand.New(rand.NewSource(1))
	result, err := expr.Roll(rng)
	if err != nil {
		t.Fatalf("Roll returned error: %v", err)
	}
	out, err := Format(result, true)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}
	if out == "" {
		t.Fatal("expected non-empty JSON output")
	}
}
