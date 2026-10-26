package vote

import (
	"testing"

	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
)

// board builds entries with competition ranks from scores sorted high to low.
func board(scores ...int32) []model.LeaderboardEntry {
	entries := make([]model.LeaderboardEntry, len(scores))
	for i, s := range scores {
		rank := int32(i + 1)
		if i > 0 && s == scores[i-1] {
			rank = entries[i-1].Rank
		}
		entries[i] = model.LeaderboardEntry{Rank: rank, Score: s}
	}
	return entries
}

func payouts(entries []model.LeaderboardEntry) []int64 {
	out := make([]int64, len(entries))
	for i, e := range entries {
		out[i] = e.Payout
	}
	return out
}

func TestSplitPrize(t *testing.T) {
	tests := []struct {
		name   string
		pool   int64
		scores []int32
		want   []int64
	}{
		{"one scorer takes all", 50000, []int32{3, 0, 0}, []int64{50000, 0, 0}},
		{"second scorer joins", 50000, []int32{3, 1}, []int64{37500, 12500}},
		{"three scorers", 50000, []int32{3, 1, 1}, []int64{30000, 10000, 10000}},
		{"two thirds share", 50000, []int32{3, 2, 1, 1}, []int64{21431, 14285, 7142, 7142}},
		{"two seconds push 4th out", 50000, []int32{3, 2, 2, 1}, []int64{21430, 14285, 14285, 0}},
		{"tie for first keeps leftover", 50000, []int32{3, 3, 3, 1}, []int64{16666, 16666, 16666, 0}},
		{"fourth rank gets nothing", 50000, []int32{4, 3, 2, 1}, []int64{22223, 16666, 11111, 0}},
		{"nobody scored", 50000, []int32{0, 0}, []int64{0, 0}},
		{"no players", 50000, nil, []int64{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := board(tt.scores...)
			SplitPrize(tt.pool, entries)
			got := payouts(entries)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestSplitPrizeNeverPaysMoreThanPool(t *testing.T) {
	for _, scores := range [][]int32{{7, 5, 5, 5, 2}, {1, 1, 1, 1, 1}, {9, 8, 7}, {10, 1}} {
		entries := board(scores...)
		SplitPrize(99_999, entries)
		var sum int64
		for _, e := range entries {
			sum += e.Payout
		}
		if sum > 99_999 {
			t.Fatalf("scores %v paid %d, more than the pool", scores, sum)
		}
	}
}
