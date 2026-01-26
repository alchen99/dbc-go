package main

import (
	"context"
	"fmt"
	"log"

	"os"
	"time"

	"github.com/alchen99/dbc-go/common"
	"github.com/alchen99/dbc-go/helpers"
	"github.com/alchen99/dbc-go/instructions"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/mew-sh/dotenv"
)

func monitorBondingCurveGraduation() {
	env, err := dotenv.Read(".env")
	if err != nil {
		log.Fatal(err)
	}

	rpcClient := rpc.New(env["RPC_URL"])

	// Get TOKEN_MINT_ADDRESS from command line argument, fallback to .env
	tokenMintStr := ""
	if len(os.Args) > 1 {
		tokenMintStr = os.Args[1]
	} else {
		tokenMintStr = env["TOKEN_MINT_ADDRESS"]
	}

	if tokenMintStr == "" {
		log.Fatal("TOKEN_MINT_ADDRESS is required as an argument or in .env")
	}

	tokenMint := solana.MustPublicKeyFromBase58(tokenMintStr)
	ctx := context.Background()

	fmt.Printf("Monitoring bonding curve progress for token: %s\n", tokenMintStr)

	var poolAddress solana.PublicKey
	var configAddress solana.PublicKey
	var lastProgress float64 = -1

	// Strategy: Find the Pool where the token is the BaseMint
	// Pool Account Discriminator: [213, 224, 5, 209, 98, 69, 119, 92]
	// BaseMint Offset: 136
	poolDiscriminator := []byte{213, 224, 5, 209, 98, 69, 119, 92}

	// Initial fetch to find the pool address
	pools, err := rpcClient.GetProgramAccountsWithOpts(
		ctx,
		solana.MustPublicKeyFromBase58(common.DbcProgramID),
		&rpc.GetProgramAccountsOpts{
			Filters: []rpc.RPCFilter{
				{
					Memcmp: &rpc.RPCFilterMemcmp{
						Offset: 0,
						Bytes:  poolDiscriminator,
					},
				},
				{
					Memcmp: &rpc.RPCFilterMemcmp{
						Offset: 136,
						Bytes:  tokenMint.Bytes(),
					},
				},
			},
		},
	)

	if err != nil {
		log.Fatalf("Failed to search for pools: %v", err)
	}

	if len(pools) == 0 {
		log.Fatal("Could not find any Meteora DBC pool where this token is the BaseMint.")
	}

	poolAccount := pools[0]
	poolAddress = poolAccount.Pubkey
	fmt.Println("Found Pool Address:", poolAddress.String())

	// Loop until graduated
	for {
		// Fetch current pool state
		resp, err := rpcClient.GetAccountInfo(ctx, poolAddress)
		if err != nil {
			fmt.Printf("Error fetching pool info: %v. Retrying...\n", err)
			time.Sleep(5 * time.Second)
			continue
		}

		poolData := resp.Value.Data.GetBinary()
		pool, err := helpers.DeserializePool(poolData)
		if err != nil {
			log.Fatalf("Failed to deserialize pool: %v", err)
		}
		configAddress = pool.Config

		// Fetch Pool Config
		poolConfig, err := instructions.GetPoolConfig(ctx, configAddress, rpcClient)
		if err != nil {
			log.Fatalf("Failed to get pool config: %v", err)
		}

		// Calculate Progress
		progress := float64(pool.QuoteReserve) / float64(poolConfig.MigrationQuoteThreshold) * 100

		if progress != lastProgress {
			fmt.Printf("[%s] Progress: %.2f%% | Quote Reserve: %d / %d\n",
				time.Now().Format("15:04:05"),
				progress,
				pool.QuoteReserve,
				poolConfig.MigrationQuoteThreshold,
			)
			lastProgress = progress
		}

		// Check if migrated
		if pool.IsMigrated == 1 {
			fmt.Println("Token has officially migrated (IsMigrated = 1)!")
			break
		}

		if progress >= 100 {
			fmt.Println("Bonding curve reached 100%. Waiting for migration transaction...")
		}

		time.Sleep(5 * time.Second)
	}

	// Token has graduated, now find where it went
	fmt.Println("Searching for graduated pool address...")
	findGraduatedPool(ctx, rpcClient, tokenMint)
}

func findGraduatedPool(ctx context.Context, rpcClient *rpc.Client, tokenMint solana.PublicKey) {
	// 1. Try DAMM V2 (DLMM)
	// Typical DLMM layout: Disc(8), Config(32), TokenX(32), TokenY(32)
	// We check offset 40 and 72 for the mint.
	v2ProgramID := solana.MustPublicKeyFromBase58(common.DammV2ProgramID)

	for _, offset := range []uint64{40, 72} {
		pools, err := rpcClient.GetProgramAccountsWithOpts(
			ctx,
			v2ProgramID,
			&rpc.GetProgramAccountsOpts{
				Filters: []rpc.RPCFilter{
					{
						Memcmp: &rpc.RPCFilterMemcmp{
							Offset: offset,
							Bytes:  tokenMint.Bytes(),
						},
					},
				},
			},
		)
		if err == nil && len(pools) > 0 {
			fmt.Printf("SUCCESS: Token graduated to a DAMM v2 (DLMM) pool!\n")
			fmt.Printf("Pool Address: %s\n", pools[0].Pubkey.String())
			return
		}
	}

	// 2. Try DAMM V1 (Standard AMM)
	// Typical AMM layout: Disc(8), TokenA(32), TokenB(32)
	// We check offset 8 and 40 for the mint.
	v1ProgramID := solana.MustPublicKeyFromBase58(common.DammV1ProgramID)

	for _, offset := range []uint64{8, 40} {
		pools, err := rpcClient.GetProgramAccountsWithOpts(
			ctx,
			v1ProgramID,
			&rpc.GetProgramAccountsOpts{
				Filters: []rpc.RPCFilter{
					{
						Memcmp: &rpc.RPCFilterMemcmp{
							Offset: offset,
							Bytes:  tokenMint.Bytes(),
						},
					},
				},
			},
		)
		if err == nil && len(pools) > 0 {
			fmt.Printf("SUCCESS: Token graduated to a DAMM v1 (Dynamic AMM) pool!\n")
			fmt.Printf("Pool Address: %s\n", pools[0].Pubkey.String())
			return
		}
	}

	fmt.Println("Graduation detected on-chain but could not locate the new pool in DAMM v1 or v2 programs yet.")
	fmt.Println("It might still be in the process of initialization.")
}

func main() {
	monitorBondingCurveGraduation()
}
