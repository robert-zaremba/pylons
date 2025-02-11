package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/Pylons-tech/pylons/x/pylons/keeper"
	"github.com/Pylons-tech/pylons/x/pylons/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestAfterEpochEndWithDeligators() {
	k := suite.k
	sk := suite.stakingKeeper
	ctx := suite.ctx
	require := suite.Require()
	bk := suite.bankKeeper
	ak := suite.accountKeeper

	srv := keeper.NewMsgServerImpl(k)
	wctx := sdk.WrapSDKContext(ctx)

	/*
		* amountToPay := refers to the recipe amount
		* ``
		*	form this amount we will calculate the reward that needs to be distributed to
		*	the delegator of the block
		* ``
		* creator  := create address will be used to create cookbook / recipe
		* executor := will be used to execute recipe, as creator and executor cannot be same
		*
		* upon execution of recipe we have a defined fee,
		* i.e. DefaultRecipeFeePercentage (Set at 0.1 or 10%)
		*
		* feeCollectorAddr := address of you fee collector module
		* this modules receives the fee deducted during recipe execution
		*
		* Pre Req:
		*	1. Create Cookbook
		*	2. Create Recipe
		*	3. Execute Recipe
		*
		*
		* this test case will verify that correct amount of rewards are divided amongst delegator
		*
		* 1. Get `delegator amount percentage` that need to be distributed
		* 2. Calculate delegator reward amount for distributed
		* 3. Get balance of delegator before sending reward
		* 4. Distribute reward amongst delegator
		* 5. Query for balance of delegator again to get update balance after sending rewards
		* 6. Compare balance from step 3 with step 5,
			* 6.1 New balance must be equivalent with the old balance
				plus reward amount calculated in step 2
		*
		* Criteria: In case the balances must match , i.e. (balance before distribution of reward
		*			+ the reward amount) == balance after distribution
		*
	*/

	amountToPay := sdk.NewCoins(sdk.NewCoin(types.PylonsCoinDenom, sdk.NewInt(100)))
	creator := types.GenTestBech32FromString("test")
	executor := types.GenTestBech32FromString("executor")
	feeCollectorAddr := ak.GetModuleAddress(types.FeeCollectorName)

	// Required to disable app check enforcement to make an account
	types.UpdateAppCheckFlagTest(types.FlagTrue)

	// create an account for the executor as their account in pylons is required
	srv.CreateAccount(wctx, &types.MsgCreateAccount{
		Creator: executor,
	})

	// enable the app check enforcement again
	types.UpdateAppCheckFlagTest(types.FlagFalse)

	// making an instance of cookbook
	cookbookMsg := &types.MsgCreateCookbook{
		Creator:      creator,
		Id:           "testCookbookID",
		Name:         "testCookbookName",
		Description:  "descdescdescdescdescdesc",
		Version:      "v0.0.1",
		SupportEmail: "test@email.com",
		Enabled:      true,
	}
	// creating a cookbook
	_, err := srv.CreateCookbook(sdk.WrapSDKContext(suite.ctx), cookbookMsg)
	// must not throw any error
	require.NoError(err)
	// making an instance of cookbook
	recipeMsg := &types.MsgCreateRecipe{
		Creator:       creator,
		CookbookId:    "testCookbookID",
		Id:            "testRecipeID",
		Name:          "recipeName",
		Description:   "descdescdescdescdescdesc",
		Version:       "v0.0.1",
		BlockInterval: 10,
		CostPerBlock:  sdk.Coin{Denom: "test", Amount: sdk.ZeroInt()},
		CoinInputs:    []types.CoinInput{{Coins: amountToPay}},
		Enabled:       true,
	}
	// creating a recipe
	_, err = srv.CreateRecipe(sdk.WrapSDKContext(suite.ctx), recipeMsg)
	require.NoError(err)

	// create only one pendingExecution
	msgExecution := &types.MsgExecuteRecipe{
		Creator:         executor,
		CookbookId:      "testCookbookID",
		RecipeId:        "testRecipeID",
		CoinInputsIndex: 0,
		ItemIds:         nil,
	}

	// fund account of executer to execute recipe
	suite.FundAccount(suite.ctx, sdk.MustAccAddressFromBech32(executor), amountToPay)

	// execute a recipe
	resp, err := srv.ExecuteRecipe(sdk.WrapSDKContext(suite.ctx), msgExecution)
	require.NoError(err)

	// manually trigger complete execution - simulate endBlocker
	pendingExecution := k.GetPendingExecution(ctx, resp.Id)
	execution, _, _, err := k.CompletePendingExecution(suite.ctx, pendingExecution)
	require.NoError(err)
	k.ActualizeExecution(ctx, execution)

	// verify execution completion and that requester has no balance left,
	// also pay and fee are transfered to cookbook owner and fee collector module
	_ = bk.SpendableCoins(ctx, sdk.MustAccAddressFromBech32(executor))
	_ = bk.SpendableCoins(ctx, sdk.MustAccAddressFromBech32(creator))
	_ = bk.SpendableCoins(ctx, feeCollectorAddr)

	// get reward distribution percentages
	distrPercentages := k.GetValidatorRewardsDistributionPercentages(ctx, sk)
	// get the balance of the feeCollector moduleAcc
	rewardsTotalAmount := bk.SpendableCoins(ctx, k.FeeCollectorAddress())
	// calculate delegator rewards
	delegatorsRewards := k.CalculateValidatorRewardsHelper(distrPercentages, rewardsTotalAmount)
	delegatorMap := map[string]sdk.Coins{}
	balances := sdk.Coins{}
	// checking if delegator rewards are not nil
	if delegatorsRewards != nil {
		// looping through delegators to get their old balance
		for _, reward := range delegatorsRewards {
			// looping through amount type of sdk.coins to get every amount and denom
			for _, val := range reward.Coins {
				oldBalance := suite.bankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(reward.Address), val.Denom)
				// Appending old balance in balances so we can compare it later on with updated balance
				balances = append(balances, oldBalance.Add(val))
			}
			delegatorMap[reward.Address] = balances

		}
		// sending rewards to delegators
		k.SendRewards(ctx, delegatorsRewards)
		for address, updatedAmount := range delegatorMap {
			// looping through updated amount type of sdk.coins to get every amount and denom
			for _, val := range updatedAmount {
				// balance after reward distribution
				newBalance := suite.bankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(address), val.Denom).Amount
				// balance calculated on line#164
				balanceToEqual := val.Amount // this amount is equal to the balance of user before reward distribution + reward to be distributed
				// Comparing balances of delegator before and after reward distribution
				require.Equal(balanceToEqual, newBalance)
			}

		}

	}
}

// Test Case For After Epoch End Fuction With Case No deligators
func (suite *IntegrationTestSuite) TestAfterEpochEndNoDeligators() {
	k := suite.k
	ctx := suite.ctx
	require := suite.Require()
	bk := suite.bankKeeper
	ak := suite.accountKeeper

	srv := keeper.NewMsgServerImpl(k)
	wctx := sdk.WrapSDKContext(ctx)

	amountToPay := sdk.NewCoins(sdk.NewCoin(types.PylonsCoinDenom, sdk.NewInt(100)))
	creator := types.GenTestBech32FromString("test")
	executor := types.GenTestBech32FromString("executor")
	feeCollectorAddr := ak.GetModuleAddress(types.FeeCollectorName)

	types.UpdateAppCheckFlagTest(types.FlagTrue)

	srv.CreateAccount(wctx, &types.MsgCreateAccount{
		Creator: executor,
	})

	types.UpdateAppCheckFlagTest(types.FlagFalse)
	cookbookMsg := &types.MsgCreateCookbook{
		Creator:      creator,
		Id:           "testCookbookID",
		Name:         "testCookbookName",
		Description:  "descdescdescdescdescdesc",
		Version:      "v0.0.1",
		SupportEmail: "test@email.com",
		Enabled:      true,
	}
	_, err := srv.CreateCookbook(sdk.WrapSDKContext(suite.ctx), cookbookMsg)
	require.NoError(err)
	recipeMsg := &types.MsgCreateRecipe{
		Creator:       creator,
		CookbookId:    "testCookbookID",
		Id:            "testRecipeID",
		Name:          "recipeName",
		Description:   "descdescdescdescdescdesc",
		Version:       "v0.0.1",
		BlockInterval: 10,
		CostPerBlock:  sdk.Coin{Denom: "test", Amount: sdk.ZeroInt()},
		CoinInputs:    []types.CoinInput{{Coins: amountToPay}},
		Enabled:       true,
	}
	_, err = srv.CreateRecipe(sdk.WrapSDKContext(suite.ctx), recipeMsg)
	require.NoError(err)

	// create only one pendingExecution
	msgExecution := &types.MsgExecuteRecipe{
		Creator:         executor,
		CookbookId:      "testCookbookID",
		RecipeId:        "testRecipeID",
		CoinInputsIndex: 0,
		ItemIds:         nil,
	}

	// give coins to requester
	suite.FundAccount(suite.ctx, sdk.MustAccAddressFromBech32(executor), amountToPay)

	resp, err := srv.ExecuteRecipe(sdk.WrapSDKContext(suite.ctx), msgExecution)
	require.NoError(err)

	// manually trigger complete execution - simulate endBlocker
	pendingExecution := k.GetPendingExecution(ctx, resp.Id)
	execution, _, _, err := k.CompletePendingExecution(suite.ctx, pendingExecution)
	require.NoError(err)
	k.ActualizeExecution(ctx, execution)

	// verify execution completion and that requester has no balance left,
	// also pay and fee are transfered to cookbook owner and fee collector module
	_ = bk.SpendableCoins(ctx, sdk.MustAccAddressFromBech32(executor))
	_ = bk.SpendableCoins(ctx, sdk.MustAccAddressFromBech32(creator))
	_ = bk.SpendableCoins(ctx, feeCollectorAddr)

	// get the balance of the feeCollector moduleAcc
	rewardsTotalAmount := bk.SpendableCoins(ctx, k.FeeCollectorAddress())
	validatorRewards := k.CalculateValidatorRewardsHelper(nil, rewardsTotalAmount)
	delegatorMap := map[string]sdk.Coins{}
	balances := sdk.Coins{}
	if len(validatorRewards) == 0 {
		// In this Case No loop will be executed because we have no deligators to send reward
		// looping through delegators to get their old balance
		for _, reward := range validatorRewards {
			// looping through amount type of sdk.coins to get every amount and denom
			for _, val := range reward.Coins {
				oldBalance := suite.bankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(reward.Address), val.Denom)
				// Appending old balance in balances so we can compare it later on with updated balance
				balances = append(balances, oldBalance.Add(val))
			}
			delegatorMap[reward.Address] = balances

		}
		// sending rewards to delegators
		k.SendRewards(ctx, validatorRewards)
		for address, updatedAmount := range delegatorMap {
			// looping through updated amount type of sdk.coins to get every amount and denom
			for _, val := range updatedAmount {
				newBalance := suite.bankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(address), val.Denom)
				// Comparing updated Amount with new new Blanace both should  be equal
				require.Equal(val.Amount.Int64(), newBalance.Amount.Int64())
			}
		}
	}
}

func (suite *IntegrationTestSuite) TestAfterEpochEnd() {
	k := suite.k
	sk := suite.stakingKeeper
	ctx := suite.ctx
	require := suite.Require()
	bk := suite.bankKeeper
	ak := suite.accountKeeper

	srv := keeper.NewMsgServerImpl(k)
	wctx := sdk.WrapSDKContext(ctx)
	type Account struct {
		address       string
		name          string
		coins         sdk.Coins
		expectedCoins sdk.Coins
	}
	tests := []struct {
		desc                    string
		accounts                []Account
		validatorBalance        int64
		updatedValidatorBalance int64
		err                     error
	}{
		{
			desc: "Epoch end with 1 delegator and 1 validator",
			accounts: []Account{
				{
					address: types.GenTestBech32FromString("creator1"),
					name:    "test",
					coins: sdk.Coins{
						sdk.Coin{Denom: types.PylonsCoinDenom, Amount: sdk.NewInt(0)},
					},
					expectedCoins: sdk.Coins{
						sdk.Coin{Denom: types.PylonsCoinDenom, Amount: sdk.NewInt(90)},
					},
				},
				{
					address: types.GenTestBech32FromString("executer2"),
					name:    "test3",
					coins: sdk.Coins{
						sdk.Coin{Denom: types.PylonsCoinDenom, Amount: sdk.NewInt(100)},
					},
					expectedCoins: sdk.Coins{
						sdk.Coin{Denom: types.PylonsCoinDenom, Amount: sdk.NewInt(0)},
					},
				},
				{
					address: types.GenTestBech32FromString("holder3"),
					name:    "holder",
					coins: sdk.Coins{
						sdk.Coin{Denom: types.StakingCoinDenom, Amount: sdk.NewInt(100)},
					},
					expectedCoins: sdk.Coins{
						sdk.Coin{Denom: types.PylonsCoinDenom, Amount: sdk.NewInt(9)},
					},
				},
			},
			validatorBalance:        0,
			updatedValidatorBalance: 1,
		},
		{
			desc: "Epoch end with 2 delegators and 1 validator",
			accounts: []Account{
				{
					address: types.GenTestBech32FromString("creator2"),
					name:    "test2",
					coins: sdk.Coins{
						sdk.Coin{Denom: types.PylonsCoinDenom, Amount: sdk.NewInt(0)},
					},
					expectedCoins: sdk.Coins{
						sdk.Coin{Denom: types.PylonsCoinDenom, Amount: sdk.NewInt(90)},
					},
				},
				{
					address: types.GenTestBech32FromString("executer3"),
					name:    "test4",
					coins: sdk.Coins{
						sdk.Coin{Denom: types.PylonsCoinDenom, Amount: sdk.NewInt(100)},
					},
					expectedCoins: sdk.Coins{
						sdk.Coin{Denom: types.PylonsCoinDenom, Amount: sdk.NewInt(0)},
					},
				},
				{
					address: types.GenTestBech32FromString("holder5"),
					name:    "holder",
					coins: sdk.Coins{
						sdk.Coin{Denom: types.StakingCoinDenom, Amount: sdk.NewInt(100)},
					},
					expectedCoins: sdk.Coins{
						sdk.Coin{Denom: types.PylonsCoinDenom, Amount: sdk.NewInt(2)},
					},
				},
				{
					address: types.GenTestBech32FromString("rockholder"),
					name:    "rockholder",
					coins: sdk.Coins{
						sdk.Coin{Denom: types.StakingCoinDenom, Amount: sdk.NewInt(200)},
					},
					expectedCoins: sdk.Coins{
						sdk.Coin{Denom: types.PylonsCoinDenom, Amount: sdk.NewInt(4)},
					},
				},
			},
			validatorBalance:        1,
			updatedValidatorBalance: 2,
		},
	}
	for _, tc := range tests {
		suite.Run(tc.desc, func() {
			types.UpdateAppCheckFlagTest(types.FlagTrue)
			// create account for address
			for _, acc := range tc.accounts {
				srv.CreateAccount(wctx, &types.MsgCreateAccount{
					Creator: acc.address,
				})
				// give accounts initial coins
				suite.FundAccount(ctx, sdk.MustAccAddressFromBech32(acc.address), acc.coins)
			}
			creator := tc.accounts[0]
			executor := tc.accounts[1]
			cookbookId := tc.accounts[0].name
			recipeId := tc.accounts[1].name
			types.UpdateAppCheckFlagTest(types.FlagFalse)
			cookbookMsg := &types.MsgCreateCookbook{
				Creator:      creator.address,
				Id:           cookbookId,
				Name:         "testCookbookName",
				Description:  "descdescdescdescdescdesc",
				Version:      "v0.0.1",
				SupportEmail: "test@email.com",
				Enabled:      true,
			}
			_, err := srv.CreateCookbook(sdk.WrapSDKContext(suite.ctx), cookbookMsg)
			require.NoError(err)
			recipeMsg := &types.MsgCreateRecipe{
				Creator:       creator.address,
				CookbookId:    cookbookId,
				Id:            recipeId,
				Name:          "recipeName",
				Description:   "descdescdescdescdescdesc",
				Version:       "v0.0.1",
				BlockInterval: 10,
				CostPerBlock:  sdk.Coin{Denom: "test", Amount: sdk.ZeroInt()},
				CoinInputs: []types.CoinInput{{Coins: sdk.Coins{
					sdk.Coin{
						Denom:  types.PylonsCoinDenom,
						Amount: math.NewInt(100),
					},
				}}},
				Enabled: true,
			}
			_, err = srv.CreateRecipe(sdk.WrapSDKContext(suite.ctx), recipeMsg)
			require.NoError(err)
			// create only one pendingExecution
			msgExecution := &types.MsgExecuteRecipe{
				Creator:         executor.address,
				CookbookId:      cookbookId,
				RecipeId:        recipeId,
				CoinInputsIndex: 0,
				ItemIds:         nil,
			}
			resp, err := srv.ExecuteRecipe(sdk.WrapSDKContext(suite.ctx), msgExecution)
			require.NoError(err)
			// manually trigger complete execution - simulate endBlocker
			pendingExecution := k.GetPendingExecution(ctx, resp.Id)
			execution, _, _, err := k.CompletePendingExecution(suite.ctx, pendingExecution)
			require.NoError(err)
			k.ActualizeExecution(ctx, execution)
			delegations := sk.GetAllSDKDelegations(ctx)
			validatorBalance := make(map[string]sdk.Coin)

			// calculating total shares for out validators
			for _, delegation := range delegations {
				validatorBalance[delegation.DelegatorAddress] = bk.GetBalance(ctx, delegation.GetDelegatorAddr(), types.PylonsCoinDenom)
				require.Equal(sdk.NewCoin(types.PylonsCoinDenom, sdk.NewInt(tc.validatorBalance)), validatorBalance[delegation.DelegatorAddress])
			}
			k.AfterEpochEnd(ctx, "day", 25, sk, ak)

			// check coins of accounts after epoch end
			for _, acc := range tc.accounts {
				accBal := bk.GetBalance(ctx, sdk.MustAccAddressFromBech32(acc.address), types.PylonsCoinDenom)
				require.Equal(acc.expectedCoins, sdk.Coins{
					accBal,
				})
			}
			// checking delegation balance
			for _, delegation := range delegations {
				delegatorPylonBalance := bk.GetBalance(ctx, delegation.GetDelegatorAddr(), types.PylonsCoinDenom)
				require.Equal(sdk.NewCoin(types.PylonsCoinDenom, sdk.NewInt(tc.updatedValidatorBalance)), delegatorPylonBalance)
			}
		})
	}
}
