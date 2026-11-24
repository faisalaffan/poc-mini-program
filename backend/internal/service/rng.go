package service

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/faisalaffan/poc-mini-program/backend/internal/domain"
)

const totalBasisPoints = 10000

// WeightedRNG mendefinisikan pemilihan hadiah berbasis CSPRNG di server.
type WeightedRNG interface {
	Pick(tiers []domain.RewardTier) (domain.RewardTier, error)
}

type CryptoWeightedRNG struct{}

func NewCryptoWeightedRNG() *CryptoWeightedRNG {
	return &CryptoWeightedRNG{}
}

func (r *CryptoWeightedRNG) Pick(tiers []domain.RewardTier) (domain.RewardTier, error) {
	if len(tiers) == 0 {
		return domain.RewardTier{}, fmt.Errorf("reward tiers must not be empty")
	}

	sum := 0
	for _, tier := range tiers {
		if tier.WeightBPS <= 0 {
			return domain.RewardTier{}, fmt.Errorf("tier %s has non-positive weight_bps %d", tier.Code, tier.WeightBPS)
		}
		sum += tier.WeightBPS
	}
	if sum != totalBasisPoints {
		return domain.RewardTier{}, fmt.Errorf("total weight_bps must equal %d, got %d", totalBasisPoints, sum)
	}

	n, err := rand.Int(rand.Reader, big.NewInt(totalBasisPoints))
	if err != nil {
		return domain.RewardTier{}, fmt.Errorf("crypto rand int: %w", err)
	}

	return selectByRoll(tiers, int(n.Int64())), nil
}

func selectByRoll(tiers []domain.RewardTier, roll int) domain.RewardTier {
	cumulative := 0
	for _, tier := range tiers {
		cumulative += tier.WeightBPS
		if roll < cumulative {
			return tier
		}
	}
	return tiers[len(tiers)-1]
}
