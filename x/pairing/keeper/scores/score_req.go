package scores

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
)

// ScoreReq is an interface for pairing requirement scoring
type ScoreReq interface {
	// Init() initializes the ScoreReq object and returns whether it's active
	Init(policy planstypes.Policy) bool
	// Score() calculates a provider's score according to the requirement
	Score(stakeEntry epochstoragetypes.StakeEntry) sdk.Uint
	// GetName returns the unique name of the ScoreReq implementation
	GetName() string
	// Equal compares two ScoreReq objects
	Equal(other ScoreReq) bool
	// GetReqForSlot gets ScoreReq for slot by its index
	GetReqForSlot(policy planstypes.Policy, slotIdx int) ScoreReq
}