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
	if expr.Count != 3 || expr.Sides != 6 || expr.Modifier != 2 {
		t.Fatalf("unexpected expression: %+v", expr)
	}
}

func TestParseKeepHighest(t *testing.T) {
	expr, err := Parse("4d6kh3")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if expr.KeepHighest != 3 {
		t.Fatalf("expected KeepHighest 3, got %d", expr.KeepHighest)
	}
}

func TestParsePercentile(t *testing.T) {
	expr, err := Parse("d%")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if expr.Count != 1 || expr.Sides != 100 {
		t.Fatalf("unexpected expression: %+v", expr)
	}
}

func TestParseInvalid(t *testing.T) {
	if _, err := Parse("not-dice"); err == nil {
		t.Fatal("expected an error for invalid notation")
	}
}

func TestParseKeepExceedsCount(t *testing.T) {
	if _, err := Parse("2d6kh3"); err == nil {
		t.Fatal("expected an error when keep count exceeds dice count")
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
	if len(result.Rolls) != 2 {
		t.Fatalf("expected 2 rolls, got %d", len(result.Rolls))
	}
	want := result.Rolls[0] + result.Rolls[1] + 1
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
	if len(result.Kept) != 3 || len(result.Dropped) != 1 {
		t.Fatalf("expected 3 kept and 1 dropped, got kept=%v dropped=%v", result.Kept, result.Dropped)
	}
	sum := 0
	for _, v := range result.Kept {
		sum += v
	}
	if sum != result.Total {
		t.Fatalf("total = %d, want sum of kept %d", result.Total, sum)
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
