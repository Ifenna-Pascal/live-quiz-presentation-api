package vote

import "github.com/ifenna-pascal/interactive-presentation/internal/constant/model"

// PaidRanks is how many ranks share the prize.
const PaidRanks = 3

// SplitPrize sets each entry's Payout.
//
// Players ranked 1st to 3rd with a score above 0 share the pool in proportion
// to their scores. Ranks come from competition ranking (1, 2, 2, 4): players
// with the same score share a rank, so two players can both be 2nd or 3rd
// and they get the same amount. Everyone else gets 0.
//
// Amounts are rounded down. Whatever is left from rounding goes to the leader
// when exactly one player is ranked 1st; with a tie for 1st it is not paid
// out, so tied players always receive equal amounts.
//
// entries must carry competition ranks (as the leaderboard query returns).
// The function runs on every leaderboard read, so the split follows the
// scores live: one scorer gets everything, and the pool is shared as others
// start scoring.
func SplitPrize(pool int64, entries []model.LeaderboardEntry) {
	var total int64
	var leaders int
	for i := range entries {
		entries[i].Payout = 0
		if isPaid(entries[i]) {
			total += int64(entries[i].Score)
			if entries[i].Rank == 1 {
				leaders++
			}
		}
	}
	if pool <= 0 || total == 0 {
		return
	}

	var paid int64
	leader := -1
	for i := range entries {
		if !isPaid(entries[i]) {
			continue
		}
		entries[i].Payout = pool * int64(entries[i].Score) / total
		paid += entries[i].Payout
		if entries[i].Rank == 1 {
			leader = i
		}
	}
	if leaders == 1 {
		entries[leader].Payout += pool - paid
	}
}

func isPaid(e model.LeaderboardEntry) bool {
	return e.Score > 0 && e.Rank >= 1 && e.Rank <= PaidRanks
}
