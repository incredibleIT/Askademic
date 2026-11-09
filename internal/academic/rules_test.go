package academic

import (
	"math"
	"testing"
)

func TestGPABands(t *testing.T) {
	r := DefaultRules()
	cases := []struct{ score, want float64 }{
		{95, 4.0}, {89, 3.7}, {83, 3.3}, {79, 3.0}, {76, 2.7},
		{73, 2.3}, {69, 2.0}, {65, 1.5}, {61, 1.0}, {59, 0}, {0, 0},
	}
	for _, c := range cases {
		if got := r.GPA(c.score); got != c.want {
			t.Errorf("GPA(%v)=%v want %v", c.score, got, c.want)
		}
	}
}

func TestTotalScoreWeights(t *testing.T) {
	r := DefaultRules()
	if got := r.TotalScore(80, 90); math.Abs(got-87) > 0.001 {
		t.Fatalf("TotalScore(80,90)=%v want 87", got)
	}
}
