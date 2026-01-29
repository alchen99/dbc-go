package main

import (
	"bytes"
	"context"
	"crypto/sha256"
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
	findGraduatedPool(ctx, rpcClient, tokenMint, poolAddress)
}

func findGraduatedPool(ctx context.Context, rpcClient *rpc.Client, tokenMint solana.PublicKey, dbcPoolAddress solana.PublicKey) {
	// 1. Try Migration Transaction Parsing (Most Reliable)
	fmt.Println("Checking migration transactions...")
	if found := findGraduatedPoolByMigrationTx(ctx, rpcClient, dbcPoolAddress); found {
		return
	}

	// 2. Try DAMM V2 (DLMM)
	// Typical DLMM layout: Disc(8), Config(32), TokenX(32), TokenY(32)
	// We check offset 40 and 72 for the mint.
	dlmmProgramID := solana.MustPublicKeyFromBase58(common.DlmmProgramID)

	for _, offset := range []uint64{40, 72} {
		pools, err := rpcClient.GetProgramAccountsWithOpts(
			ctx,
			dlmmProgramID,
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
			fmt.Printf("SUCCESS (GPA): Token graduated to a DLMM pool!\n")
			fmt.Printf("Pool Address: %s\n", pools[0].Pubkey.String())
			return
		}
	}

	// 3. Try CP-Swap (Commerce Partners)
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
			fmt.Printf("SUCCESS (GPA): Token graduated to a CP-Swap pool!\n")
			fmt.Printf("Pool Address: %s\n", pools[0].Pubkey.String())
			return
		}
	}

	// 4. Try DAMM V1 (Standard AMM)
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

	fmt.Println("Graduation detected on-chain but could not locate the new pool in DAMM v1, v2 or DLMM programs yet.")
	fmt.Println("It might still be in the process of initialization.")
}

func findGraduatedPoolByMigrationTx(ctx context.Context, rpcClient *rpc.Client, dbcPoolAddress solana.PublicKey) bool {
	limit := 20
	// Fetch recent signatures for the DBC pool
	signatures, err := rpcClient.GetSignaturesForAddressWithOpts(
		ctx,
		dbcPoolAddress,
		&rpc.GetSignaturesForAddressOpts{
			Limit: &limit,
		},
	)
	if err != nil {
		fmt.Printf("Failed to get signatures for DBC pool: %v\n", err)
		return false
	}

	dbcProgramID := solana.MustPublicKeyFromBase58(common.DbcProgramID)

	// Calculate discriminators for migration instructions
	// Anchor discriminator = sha256("global:<name>")[:8]
	calcDisc := func(name string) []byte {
		h := sha256.New()
		h.Write([]byte(fmt.Sprintf("global:%s", name)))
		return h.Sum(nil)[:8]
	}

	migrationDammV2Disc := calcDisc("migration_damm_v2")
	migrateMeteoraDammDisc := calcDisc("migrate_meteora_damm")

	for _, sig := range signatures {
		maxVersion := uint64(0)
		txResp, err := rpcClient.GetTransaction(
			ctx,
			sig.Signature,
			&rpc.GetTransactionOpts{
				MaxSupportedTransactionVersion: &maxVersion,
				Encoding:                       solana.EncodingBase64,
			},
		)
		if err != nil || txResp == nil {
			continue
		}

		tx, err := txResp.Transaction.GetTransaction()
		if err != nil {
			continue
		}

		// Look for instructions invoking the DBC program
		for _, inst := range tx.Message.Instructions {
			programID := tx.Message.AccountKeys[inst.ProgramIDIndex]
			if !programID.Equals(dbcProgramID) {
				continue
			}

			// Check if it's a migration instruction
			if len(inst.Data) < 8 {
				continue
			}

			disc := inst.Data[:8]
			if bytes.Equal(disc, migrationDammV2Disc) {
				// migration_damm_v2: New Pool is typically at index 4 of instruction accounts
				if len(inst.Accounts) > 4 {
					poolIndex := inst.Accounts[4]
					poolAddress := tx.Message.AccountKeys[poolIndex]
					fmt.Printf("SUCCESS (TX): Found graduated pool via migration_damm_v2!\n")
					fmt.Printf("Pool Address: %s\n", poolAddress.String())
					return true
				}
			} else if bytes.Equal(disc, migrateMeteoraDammDisc) {
				// migrate_meteora_damm: We'll look for an account owned by a Meteora program
				// or assume similar positioning if possible.
				// Based on common patterns, it's usually one of the early writable accounts.
				for _, accIdx := range inst.Accounts {
					accAddr := tx.Message.AccountKeys[accIdx]
					// Check if this account is owned by a Meteora program (requires additional RPC call or heuristic)
					// For now, if we found the instruction, we'll print the first candidate or all writable ones.
					fmt.Printf("SUCCESS (TX): Found migration_meteora_damm instruction!\n")
					fmt.Printf("Candidate Pool Address: %s\n", accAddr.String())
					return true
				}
			}
		}
	}
	return false
}

// func main() {
// 	monitorBondingCurveGraduation()
// }
