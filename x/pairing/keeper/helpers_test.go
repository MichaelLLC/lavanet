package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/testutil/common"
	testutil "github.com/lavanet/lava/v5/testutil/keeper"
	dualstakingtypes "github.com/lavanet/lava/v5/x/dualstaking/types"
	epochstoragetypes "github.com/lavanet/lava/v5/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	planstypes "github.com/lavanet/lava/v5/x/plans/types"
	rewardstypes "github.com/lavanet/lava/v5/x/rewards/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
)

type tester struct {
	common.Tester
	plan planstypes.Plan
	spec spectypes.Spec
}

const (
	testBalance int64 = 1000000
	testStake   int64 = 100000
)

func newTester(t *testing.T) *tester {
	ts := &tester{Tester: *common.NewTester(t)}

	err := ts.Keepers.BankKeeper.SetBalance(ts.Ctx,
		ts.Keepers.AccountKeeper.GetModuleAddress(string(rewardstypes.ValidatorsRewardsAllocationPoolName)),
		sdk.NewCoins(sdk.NewCoin(ts.TokenDenom(), sdk.ZeroInt())))
	require.Nil(ts.T, err)

	err = ts.Keepers.BankKeeper.SetBalance(ts.Ctx,
		ts.Keepers.AccountKeeper.GetModuleAddress(string(rewardstypes.ProvidersRewardsAllocationPool)),
		sdk.NewCoins(sdk.NewCoin(ts.TokenDenom(), sdk.ZeroInt())))
	require.Nil(ts.T, err)

	ts.DisableParticipationFees()

	ts.addValidators(1)

	ts.plan = ts.AddPlan("free", common.CreateMockPlan()).Plan("free")
	ts.spec = ts.AddSpec("mock", common.CreateMockSpec()).Spec("mock")

	ts.AdvanceEpoch()

	return ts
}

func (ts *tester) addClient(count int) {
	start := len(ts.Accounts(common.CONSUMER))
	for i := 0; i < count; i++ {
		_, addr := ts.AddAccount(common.CONSUMER, start+i, testBalance)
		_, err := ts.TxSubscriptionBuy(addr, addr, ts.plan.Index, 1, false, false)
		if err != nil {
			panic("addClient: failed to buy subscription: " + err.Error())
		}
	}
}

func (ts *tester) addValidators(count int) {
	start := len(ts.Accounts(common.VALIDATOR))
	for i := 0; i < count; i++ {
		acc, _ := ts.AddAccount(common.VALIDATOR, start+i, testBalance)
		ts.TxCreateValidator(acc, math.NewInt(testBalance))
	}
}

// addProvider: with default endpoints, geolocation, moniker
func (ts *tester) addProvider(count int) error {
	d := common.MockDescription()
	return ts.addProviderExtra(count, nil, 0, d.Moniker, d.Identity, d.Website, d.SecurityContact, d.Details, ts.spec) // default: endpoints, geolocation, moniker
}

// addProvider: with default endpoints, geolocation, moniker
func (ts *tester) addProviderSpec(count int, spec string) error {
	d := common.MockDescription()
	return ts.addProviderExtra(count, nil, 0, d.Moniker, d.Identity, d.Website, d.SecurityContact, d.Details, ts.Spec(spec))
}

// addProviderGelocation: with geolocation, and default endpoints, moniker
func (ts *tester) addProviderGeolocation(count int, geolocation int32) error {
	d := common.MockDescription()
	return ts.addProviderExtra(count, nil, geolocation, d.Moniker, d.Identity, d.Website, d.SecurityContact, d.Details, ts.spec)
}

// addProviderEndpoints: with endpoints, and default geolocation, moniker
func (ts *tester) addProviderEndpoints(count int, endpoints []epochstoragetypes.Endpoint) error {
	d := common.MockDescription()
	return ts.addProviderExtra(count, endpoints, 0, d.Moniker, d.Identity, d.Website, d.SecurityContact, d.Details, ts.spec)
}

// addProviderDescription: with description, and default endpoints, geolocation
func (ts *tester) addProviderDescription(count int, moniker string, identity string, website string, securityContact string, descriptionDetails string) error {
	return ts.addProviderExtra(count, nil, 0, moniker, identity, website, securityContact, descriptionDetails, ts.spec)
}

// addProviderExtra: with mock endpoints, and preset geolocation, description details
func (ts *tester) addProviderExtra(
	count int,
	endpoints []epochstoragetypes.Endpoint,
	geoloc int32,
	moniker string,
	identity string,
	website string,
	securityContact string,
	descriptionDetails string,
	spec spectypes.Spec,
) error {
	start := len(ts.Accounts(common.PROVIDER))
	for i := 0; i < count; i++ {
		acc, addr := ts.AddAccount(common.PROVIDER, start+i, testBalance)
		err := ts.StakeProviderExtra(acc.GetVaultAddr(), addr, spec, testStake, endpoints, geoloc, moniker, identity, website, securityContact, descriptionDetails)
		if err != nil {
			return err
		}
	}
	return nil
}

// setupForPayments creates staked providers and clients with subscriptions. They can be accessed
// using ts.Account(common.PROVIDER, idx) and ts.Account(common.PROVIDER, idx) respectively.
func (ts *tester) setupForPayments(providersCount, clientsCount, providersToPair int) *tester {
	err := ts.Keepers.BankKeeper.SetBalance(ts.Ctx,
		testutil.GetModuleAddress(string(rewardstypes.ValidatorsRewardsAllocationPoolName)),
		sdk.NewCoins(sdk.NewCoin(ts.TokenDenom(), sdk.ZeroInt())))
	require.Nil(ts.T, err)

	err = ts.Keepers.BankKeeper.SetBalance(ts.Ctx,
		testutil.GetModuleAddress(string(rewardstypes.ProvidersRewardsAllocationPool)),
		sdk.NewCoins(sdk.NewCoin(ts.TokenDenom(), sdk.ZeroInt())))
	require.Nil(ts.T, err)

	ts.addValidators(1)
	if providersToPair > 0 {
		// will overwrite the default "free" plan
		ts.plan.PlanPolicy.MaxProvidersToPair = uint64(providersToPair)
		ts.AddPlan("free", ts.plan)
	}

	ts.addClient(clientsCount)
	err = ts.addProvider(providersCount)
	require.Nil(ts.T, err)

	ts.AdvanceEpoch()

	return ts
}

const (
	GreatQos = iota
	GoodQos
	BadQos
)

func (ts *tester) setupForReputation(modifyHalfLifeFactor bool) (*tester, []pairingtypes.QualityOfServiceReport) {
	ts.setupForPayments(0, 1, 5) // 0 providers, 1 client, default providers-to-pair

	greatQos := pairingtypes.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}
	goodQos := pairingtypes.QualityOfServiceReport{Latency: sdk.NewDec(3), Availability: sdk.OneDec(), Sync: sdk.NewDec(3)}
	badQos := pairingtypes.QualityOfServiceReport{Latency: sdk.NewDec(1000), Availability: sdk.OneDec(), Sync: sdk.NewDec(1000)}

	if modifyHalfLifeFactor {
		// set half life factor to be epoch time
		resQParams, err := ts.Keepers.Pairing.Params(ts.GoCtx, &pairingtypes.QueryParamsRequest{})
		require.NoError(ts.T, err)
		resQParams.Params.ReputationHalfLifeFactor = uint64(ts.EpochTimeDefault().Seconds())
		ts.Keepers.Pairing.SetParams(ts.Ctx, resQParams.Params)
	}

	// set min self delegation to zero
	resQParams2, err := ts.Keepers.Dualstaking.Params(ts.GoCtx, &dualstakingtypes.QueryParamsRequest{})
	require.NoError(ts.T, err)
	resQParams2.Params.MinSelfDelegation = sdk.NewCoin(ts.TokenDenom(), sdk.ZeroInt())
	ts.Keepers.Dualstaking.SetParams(ts.Ctx, resQParams2.Params)

	return ts, []pairingtypes.QualityOfServiceReport{greatQos, goodQos, badQos}
}

// payAndVerifyBalance performs payment and then verifies the balances
// (provider balance should increase and consumer should decrease)
// The providerRewardPerc arg is the part of the provider reward after deducting
// the delegators portion (in percentage)
func (ts *tester) payAndVerifyBalance(
	relayPayment pairingtypes.MsgRelayPayment,
	clientAddr sdk.AccAddress,
	providerVault sdk.AccAddress,
	validConsumer bool,
	validPayment bool,
	providerRewardPerc uint64,
) {
	proj, err := ts.QueryProjectDeveloper(clientAddr.String())
	if !validConsumer {
		require.NotNil(ts.T, err)
		_, err = ts.TxPairingRelayPayment(relayPayment.Creator, relayPayment.Relays[0])
		require.NotNil(ts.T, err)
		return
	}
	// else: valid consumer
	require.Nil(ts.T, err)

	sub, err := ts.QuerySubscriptionCurrent(proj.Project.Subscription)
	require.Nil(ts.T, err)
	require.NotNil(ts.T, sub.Sub)

	originalProjectUsedCu := proj.Project.UsedCu
	originalSubCuLeft := sub.Sub.MonthCuLeft

	// perform payment
	res, err := ts.TxPairingRelayPayment(relayPayment.Creator, relayPayment.Relays...)
	if !validPayment {
		if err == nil {
			require.True(ts.T, res.RejectedRelays)
			return
		}
		require.NotNil(ts.T, err)
		return
	}
	// else: valid payment
	require.Nil(ts.T, err)

	// calculate total used CU
	var totalCuUsed uint64
	var totalPaid uint64

	qosWeight := ts.Keepers.Pairing.QoSWeight(ts.Ctx)
	qosWeightComplement := sdk.OneDec().Sub(qosWeight)

	for _, relay := range relayPayment.Relays {
		cuUsed := relay.CuSum
		totalCuUsed += cuUsed
		if relay.QosReport != nil {
			score, err := relay.QosReport.ComputeQoS()
			require.Nil(ts.T, err)

			cuUsed = score.
				Mul(qosWeight).
				Add(qosWeightComplement).
				MulInt64(int64(cuUsed)).
				TruncateInt().
				Uint64()
		}

		totalPaid += cuUsed
	}

	providerReward := (totalPaid * providerRewardPerc) / 100

	// verify each project balance
	// (project used-cu should increase and respective subscription cu-left should decrease)
	proj, err = ts.QueryProjectDeveloper(clientAddr.String())
	require.Nil(ts.T, err)
	require.Equal(ts.T, originalProjectUsedCu+totalCuUsed, proj.Project.UsedCu)
	sub, err = ts.QuerySubscriptionCurrent(proj.Project.Subscription)
	require.Nil(ts.T, err)
	require.NotNil(ts.T, sub.Sub)
	require.Equal(ts.T, originalSubCuLeft-totalCuUsed, sub.Sub.MonthCuLeft)
	timeToExpiry := time.Unix(int64(sub.Sub.MonthExpiryTime), 0)
	durLeft := sub.Sub.DurationLeft
	if timeToExpiry.After(ts.Ctx.BlockTime()) && relayPayment.DescriptionString == exactConst {
		ts.AdvanceTimeHours(timeToExpiry.Sub(ts.Ctx.BlockTime()))
		// subs only pays after blocks to save
		ts.AdvanceEpoch()
		ts.AdvanceBlocks(ts.BlocksToSave() + 1)
		if durLeft > 0 {
			sub, err = ts.QuerySubscriptionCurrent(proj.Project.Subscription)
			require.Nil(ts.T, err)
			require.NotNil(ts.T, sub.Sub)
			require.Equal(ts.T, durLeft-1, sub.Sub.DurationLeft, "month expiry time: %s current time: %s", time.Unix(int64(sub.Sub.MonthExpiryTime), 0).UTC(), ts.BlockTime().UTC())
		}
	} else {
		// advance month + blocksToSave + 1 to trigger the provider monthly payment
		ts.AdvanceMonths(1)
		ts.AdvanceEpoch()
		ts.AdvanceBlocks(ts.BlocksToSave() + 1)
	}

	// verify provider's balance
	credit := sub.Sub.Credit.Amount.QuoRaw(int64(sub.Sub.DurationLeft))
	want := sdk.ZeroInt()
	if totalCuUsed != 0 {
		want = credit.MulRaw(int64(providerReward)).QuoRaw(int64(totalCuUsed))
	}

	balanceWant := ts.GetBalance(providerVault) + want.Int64()
	reward, err := ts.QueryDualstakingDelegatorRewards(providerVault.String(), relayPayment.Creator, "")
	require.Nil(ts.T, err)
	for _, reward := range reward.Rewards {
		want = want.Sub(reward.Amount.AmountOf(ts.BondDenom()))
	}
	require.True(ts.T, want.IsZero(), want)
	_, err = ts.TxDualstakingClaimRewards(providerVault.String(), relayPayment.Creator)
	require.Nil(ts.T, err)

	balance := ts.GetBalance(providerVault) + want.Int64()
	require.Equal(ts.T, balanceWant, balance)
}

// verifyRelayPayments verifies relay payments saved on-chain after getting payment
func (ts *tester) verifyRelayPayment(relaySession *pairingtypes.RelaySession, exists bool) {
	epoch := uint64(relaySession.Epoch)
	chainID := ts.spec.Name
	cu := relaySession.CuSum
	sessionID := relaySession.SessionId

	// note: assume a single client and a single provider, so these make sense:
	_, consumer := ts.GetAccount(common.CONSUMER, 0)
	_, provider := ts.GetAccount(common.PROVIDER, 0)

	project, err := ts.GetProjectForDeveloper(consumer, epoch)
	require.NoError(ts.T, err)

	found := ts.Keepers.Pairing.IsUniqueEpochSessionExists(ts.Ctx, epoch, provider, project.Index, chainID, sessionID)
	require.Equal(ts.T, exists, found)

	pec, found := ts.Keepers.Pairing.GetProviderEpochCu(ts.Ctx, epoch, provider, chainID)
	require.Equal(ts.T, exists, found)
	if exists {
		require.GreaterOrEqual(ts.T, pec.ServicedCu, cu)
	}

	pcec, found := ts.Keepers.Pairing.GetProviderConsumerEpochCu(ts.Ctx, epoch, provider, project.Index, chainID)
	require.Equal(ts.T, exists, found)
	if exists {
		require.GreaterOrEqual(ts.T, pcec.Cu, cu)
	}
}

func (ts *tester) newRelaySession(
	addr string,
	session uint64,
	cusum uint64,
	epoch uint64,
	relay uint64,
) *pairingtypes.RelaySession {
	relaySession := &pairingtypes.RelaySession{
		Provider:    addr,
		ContentHash: []byte(ts.spec.ApiCollections[0].Apis[0].Name),
		SessionId:   session,
		SpecId:      ts.spec.Name,
		CuSum:       cusum,
		Epoch:       int64(ts.EpochStart(epoch)),
		RelayNum:    relay,
	}
	return relaySession
}

func (ts *tester) isProviderFrozen(provider string, chain string) bool {
	res, err := ts.QueryPairingProvider(provider, chain)
	require.NoError(ts.T, err)
	foundChain := false
	var isFrozen bool
	for _, stakeEntry := range res.StakeEntries {
		if stakeEntry.Address == provider && stakeEntry.Chain == chain {
			foundChain = true
			isFrozen = stakeEntry.IsFrozen()
			break
		}
	}
	if !foundChain {
		require.Fail(ts.T, "provider not staked in chain", "provider: %s, chain: %s", provider, chain)
	}
	return isFrozen
}
