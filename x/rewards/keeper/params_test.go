package keeper_test

import (
	"testing"

	testkeeper "github.com/lavanet/lava/v5/testutil/keeper"
	"github.com/lavanet/lava/v5/x/rewards/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.RewardsKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}
