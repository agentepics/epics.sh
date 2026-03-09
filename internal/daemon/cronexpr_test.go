package daemon

import (
	"testing"
	"time"
)

func TestCronMatchesFiveFieldExpression(t *testing.T) {
	at := time.Date(2026, time.March, 9, 12, 30, 0, 0, time.UTC)
	if !cronMatches("30 12 * * *", at) {
		t.Fatal("expected five-field cron to match")
	}
}

func TestCronMatchesSixFieldExpression(t *testing.T) {
	at := time.Date(2026, time.March, 9, 12, 30, 10, 0, time.UTC)
	if !cronMatches("*/10 * * * * *", at) {
		t.Fatal("expected six-field cron to match")
	}
	if cronMatches("*/10 * * * * *", at.Add(5*time.Second)) {
		t.Fatal("expected six-field cron not to match non-boundary second")
	}
}

func TestCronDueBetweenSixFieldExpression(t *testing.T) {
	from := time.Date(2026, time.March, 9, 12, 30, 0, 0, time.UTC)
	to := from.Add(35 * time.Second)

	due := cronDueBetween("*/10 * * * * *", from, to)
	if len(due) != 3 {
		t.Fatalf("expected 3 due times, got %d", len(due))
	}
	expected := []int{10, 20, 30}
	for i, ts := range due {
		if ts.Second() != expected[i] {
			t.Fatalf("unexpected due time %s at index %d", ts.Format(time.RFC3339), i)
		}
	}
}
