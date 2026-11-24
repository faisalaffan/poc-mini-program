package service

import (
	"math"
	"testing"

	"github.com/faisalaffan/poc-mini-program/backend/internal/domain"
)

func TestCryptoWeightedRNG_RejectsInvalidWeights(t *testing.T) {
	rng := NewCryptoWeightedRNG()

	if _, err := rng.Pick(nil); err == nil {
		t.Fatal("expected error for empty tiers")
	}

	if _, err := rng.Pick([]domain.RewardTier{
		{Code: "A", WeightBPS: 5000},
		{Code: "B", WeightBPS: 4000},
	}); err == nil {
		t.Fatal("expected error when sum != 10000")
	}

	if _, err := rng.Pick([]domain.RewardTier{
		{Code: "A", WeightBPS: 0},
		{Code: "B", WeightBPS: 10000},
	}); err == nil {
		t.Fatal("expected error when weight_bps <= 0")
	}
}

func TestCryptoWeightedRNG_DistributionMatchesWeights(t *testing.T) {
	rng := NewCryptoWeightedRNG()
	tiers := []domain.RewardTier{
		{ID: "t1", Code: "PTS_50", Rarity: domain.RarityCommon, RewardPoints: 50, WeightBPS: 5000, DisplayOrder: 1},
		{ID: "t2", Code: "PTS_100", Rarity: domain.RarityUncommon, RewardPoints: 100, WeightBPS: 3000, DisplayOrder: 2},
		{ID: "t3", Code: "PTS_250", Rarity: domain.RarityRare, RewardPoints: 250, WeightBPS: 1400, DisplayOrder: 3},
		{ID: "t4", Code: "PTS_500", Rarity: domain.RarityEpic, RewardPoints: 500, WeightBPS: 500, DisplayOrder: 4},
		{ID: "t5", Code: "PTS_2000", Rarity: domain.RarityJackpot, RewardPoints: 2000, WeightBPS: 100, DisplayOrder: 5},
	}

	const samples = 40000
	counts := make(map[string]int, len(tiers))
	for range samples {
		picked, err := rng.Pick(tiers)
		if err != nil {
			t.Fatalf("unexpected Pick error: %v", err)
		}
		counts[picked.Code]++
	}

	for _, tier := range tiers {
		expected := float64(tier.WeightBPS) / 10000.0
		observed := float64(counts[tier.Code]) / float64(samples)
		// 5-sigma tolerance agar deterministik tanpa flaky test
		sigma := math.Sqrt(expected * (1 - expected) / float64(samples))
		tolerance := 5*sigma + 0.002
		if math.Abs(observed-expected) > tolerance {
			t.Fatalf("tier %s distribution out of bounds: expected %.4f, got %.4f (tol %.4f)",
				tier.Code, expected, observed, tolerance)
		}
	}
}
