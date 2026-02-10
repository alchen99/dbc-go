package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"

	ag_binary "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/mew-sh/dotenv"

	"github.com/alchen99/dbc-go/generated/dammv2"
	"github.com/alchen99/dbc-go/math"
)

func RemoveLiquidity() {
	env, err := dotenv.Read(".env")
	if err != nil {
		log.Printf("Warning: .env file not found, using environment variables")
	}

	rpcClient := rpc.New(getRPCURL())

	privKeyStr := env["PRIVATE_KEY"]
	if privKeyStr == "" {
		log.Fatal("PRIVATE_KEY not found in .env")
	}
	payer, err := solana.PrivateKeyFromBase58(privKeyStr)
	if err != nil {
		log.Fatalf("failed to load private key: %v", err)
	}

	// 1. Get pool address and percentage from args
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run examples/remove_liquidity.go <POOL_ADDRESS> [PERCENTAGE]")
	}
	poolAddress, err := solana.PublicKeyFromBase58(os.Args[1])
	if err != nil {
		log.Fatalf("invalid pool address: %v", err)
	}

	percentage := 100.0
	if len(os.Args) >= 3 {
		percentage, err = strconv.ParseFloat(os.Args[2], 64)
		if err != nil {
			log.Fatalf("invalid percentage: %v", err)
		}
	}

	ctx := context.Background()

	// 2. Fetch Pool Account
	fmt.Printf("Fetching pool account %s...\n", poolAddress)
	var poolAccount dammv2.PoolAccount
	err = rpcClient.GetAccountDataInto(ctx, poolAddress, &poolAccount)
	if err != nil {
		log.Fatalf("failed to fetch pool account: %v", err)
	}

	// 3. Find User's Position
	fmt.Println("Finding user's position for pool...")

	// Get all token accounts of the user
	userTokenAccounts, err := rpcClient.GetTokenAccountsByOwner(ctx, payer.PublicKey(), &rpc.GetTokenAccountsConfig{
		ProgramId: &solana.TokenProgramID,
	}, nil)
	if err != nil {
		log.Fatalf("failed to fetch user token accounts: %v", err)
	}

	userMints := make(map[solana.PublicKey]solana.PublicKey)
	for _, acc := range userTokenAccounts.Value {
		var tokenAcc token.Account
		err := rpcClient.GetAccountDataInto(ctx, acc.Pubkey, &tokenAcc)
		if err != nil {
			continue
		}
		if tokenAcc.Amount > 0 {
			userMints[tokenAcc.Mint] = acc.Pubkey
		}
	}

	// Query all positions for this pool
	positions, err := rpcClient.GetProgramAccountsWithOpts(ctx, dammv2.ProgramID, &rpc.GetProgramAccountsOpts{
		Filters: []rpc.RPCFilter{
			{
				Memcmp: &rpc.RPCFilterMemcmp{
					Offset: 0,
					Bytes:  dammv2.PositionAccountDiscriminator[:],
				},
			},
			{
				Memcmp: &rpc.RPCFilterMemcmp{
					Offset: 8,
					Bytes:  poolAddress.Bytes(),
				},
			},
		},
	})
	if err != nil {
		log.Fatalf("failed to fetch positions: %v", err)
	}

	var targetPosition *dammv2.PositionAccount
	var targetPositionAddress solana.PublicKey
	var targetPositionNftAccount solana.PublicKey

	for _, pos := range positions {
		var posAcc dammv2.PositionAccount
		err := rpcClient.GetAccountDataInto(ctx, pos.Pubkey, &posAcc)
		if err != nil {
			continue
		}

		if nftAcc, ok := userMints[posAcc.NftMint]; ok {
			targetPosition = &posAcc
			targetPositionAddress = pos.Pubkey
			targetPositionNftAccount = nftAcc
			break
		}
	}

	if targetPosition == nil {
		log.Fatal("No position found for the user in this pool.")
	}

	fmt.Printf("Found position: %s (Unlocked Liquidity: %s)\n", targetPositionAddress, targetPosition.UnlockedLiquidity.String())

	// 4. Determine Token Programs for A and B
	tokenAProgram := solana.TokenProgramID
	tokenBProgram := solana.TokenProgramID

	mintAInfo, err := rpcClient.GetAccountInfo(ctx, poolAccount.TokenAMint)
	if err == nil && mintAInfo != nil {
		tokenAProgram = mintAInfo.Value.Owner
	}
	mintBInfo, err := rpcClient.GetAccountInfo(ctx, poolAccount.TokenBMint)
	if err == nil && mintBInfo != nil {
		tokenBProgram = mintBInfo.Value.Owner
	}

	// 5. Get/Create user token accounts for A and B
	tokenAAccount, _, _ := solana.FindAssociatedTokenAddress(payer.PublicKey(), poolAccount.TokenAMint)
	tokenBAccount, _, _ := solana.FindAssociatedTokenAddress(payer.PublicKey(), poolAccount.TokenBMint)

	var ixs []solana.Instruction

	// Check if token A ATA exists
	_, err = rpcClient.GetAccountInfo(ctx, tokenAAccount)
	if err != nil {
		fmt.Println("Adding instruction to create ATA for token A...")
		ixs = append(ixs, associatedtokenaccount.NewCreateInstruction(
			payer.PublicKey(),
			payer.PublicKey(),
			poolAccount.TokenAMint,
		).Build())
	}
	// Check if token B ATA exists
	_, err = rpcClient.GetAccountInfo(ctx, tokenBAccount)
	if err != nil {
		fmt.Println("Adding instruction to create ATA for token B...")
		ixs = append(ixs, associatedtokenaccount.NewCreateInstruction(
			payer.PublicKey(),
			payer.PublicKey(),
			poolAccount.TokenBMint,
		).Build())
	}

	// 6. Build Remove Liquidity Instruction
	dummyRemIx := dammv2.NewRemoveLiquidityInstructionBuilder()
	poolAuthority, _, _ := dummyRemIx.FindPoolAuthorityAddress()
	eventAuthority, _, _ := dummyRemIx.FindEventAuthorityAddress()

	if percentage >= 100.0 {
		fmt.Println("Preparing RemoveAllLiquidity instruction...")
		ix := dammv2.NewRemoveAllLiquidityInstruction(
			0, // TokenAAmountThreshold
			0, // TokenBAmountThreshold
			poolAuthority,
			poolAddress,
			targetPositionAddress,
			tokenAAccount,
			tokenBAccount,
			poolAccount.TokenAVault,
			poolAccount.TokenBVault,
			poolAccount.TokenAMint,
			poolAccount.TokenBMint,
			targetPositionNftAccount,
			payer.PublicKey(),
			tokenAProgram,
			tokenBProgram,
			eventAuthority,
			dammv2.ProgramID,
		).Build()
		ixs = append(ixs, ix)
	} else {
		fmt.Printf("Preparing RemoveLiquidity instruction for %.2f%%...\n", percentage)

		// Calculate liquidity delta
		liqiudityBig := targetPosition.UnlockedLiquidity.BigInt()
		percBig := big.NewInt(int64(percentage * 100))
		hundredBig := big.NewInt(10000)

		deltaBig := new(big.Int).Mul(liqiudityBig, percBig)
		deltaBig.Div(deltaBig, hundredBig)

		liquidityDelta := math.Uint128FromBigInt(deltaBig)

		ix := dammv2.NewRemoveLiquidityInstruction(
			dammv2.RemoveLiquidityParameters{
				LiquidityDelta:        ag_binary.Uint128{Lo: liquidityDelta.Lo, Hi: liquidityDelta.Hi},
				TokenAAmountThreshold: 0,
				TokenBAmountThreshold: 0,
			},
			poolAuthority,
			poolAddress,
			targetPositionAddress,
			tokenAAccount,
			tokenBAccount,
			poolAccount.TokenAVault,
			poolAccount.TokenBVault,
			poolAccount.TokenAMint,
			poolAccount.TokenBMint,
			targetPositionNftAccount,
			payer.PublicKey(),
			tokenAProgram,
			tokenBProgram,
			eventAuthority,
			dammv2.ProgramID,
		).Build()
		ixs = append(ixs, ix)
	}

	// 7. Execute transaction
	fmt.Println("Assembling transaction...")
	recent, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		log.Fatalf("failed to get recent blockhash: %v", err)
	}

	tx, err := solana.NewTransaction(
		ixs,
		recent.Value.Blockhash,
		solana.TransactionPayer(payer.PublicKey()),
	)
	if err != nil {
		log.Fatalf("failed to create transaction: %v", err)
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(payer.PublicKey()) {
			return &payer
		}
		return nil
	})
	if err != nil {
		log.Fatalf("failed to sign transaction: %v", err)
	}

	fmt.Println("Sending transaction...")
	sig, err := rpcClient.SendTransaction(ctx, tx)
	if err != nil {
		log.Fatalf("failed to send transaction: %v", err)
	}
	fmt.Printf("Transaction sent: %s\n", sig)

	fmt.Println("Waiting for confirmation...")
	for i := 0; i < 60; i++ {
		time.Sleep(2 * time.Second)
		txRes, err := rpcClient.GetSignatureStatuses(ctx, false, sig)
		if err != nil || txRes == nil || len(txRes.Value) == 0 || txRes.Value[0] == nil {
			continue
		}
		if txRes.Value[0].ConfirmationStatus == rpc.ConfirmationStatusFinalized || txRes.Value[0].ConfirmationStatus == rpc.ConfirmationStatusConfirmed {
			if txRes.Value[0].Err != nil {
				log.Fatalf("Transaction failed: %v", txRes.Value[0].Err)
			}
			fmt.Printf("Transaction confirmed! https://solscan.io/tx/%s\n", sig)
			return
		}
	}
	fmt.Println("Transaction might have failed or is taking too long to confirm.")
}

// func main() {
// 	RemoveLiquidity()
// }
