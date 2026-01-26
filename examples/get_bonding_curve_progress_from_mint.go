package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/alchen99/dbc-go/common"
	"github.com/alchen99/dbc-go/helpers"
	"github.com/alchen99/dbc-go/instructions"
	"github.com/alchen99/dbc-go/math"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/mew-sh/dotenv"
	"lukechampine.com/uint128"
)

// u128ToBig converts uint128.Uint128 to *big.Int
func u128ToBig(val uint128.Uint128) *big.Int {
	hi := new(big.Int).SetUint64(val.Hi)
	lo := new(big.Int).SetUint64(val.Lo)
	hi.Lsh(hi, 64)
	return hi.Or(hi, lo)
}

func GetBondingCurveProgressFromMint() {
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

	fmt.Println("Looking for bonding curve progress for token:", tokenMintStr)
	tokenMint := solana.MustPublicKeyFromBase58(tokenMintStr)
	ctx := context.Background()

	var poolAddress solana.PublicKey
	var configAddress solana.PublicKey

	// Strategy: Find the Pool where the token is the BaseMint
	// Pool Account Discriminator: [213, 224, 5, 209, 98, 69, 119, 92]
	// BaseMint Offset: 136
	poolDiscriminator := []byte{213, 224, 5, 209, 98, 69, 119, 92}

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

	if len(pools) > 0 {
		poolAccount := pools[0]
		poolAddress = poolAccount.Pubkey
		fmt.Println("Found Pool Address:", poolAddress.String())

		// Deserialize the pool to get the Config address and current reserves
		poolData := poolAccount.Account.Data.GetBinary()
		pool, err := helpers.DeserializePool(poolData)
		if err != nil {
			log.Fatalf("Failed to deserialize pool: %v", err)
		}
		configAddress = pool.Config

		// Fetch Pool Config to get migration threshold
		poolConfig, err := instructions.GetPoolConfig(ctx, configAddress, rpcClient)
		if err != nil {
			log.Fatalf("Failed to get pool config: %v", err)
		}

		fmt.Printf("Config Address: %s\n", configAddress.String())
		fmt.Printf("Migration Quote Threshold: %d\n", poolConfig.MigrationQuoteThreshold)
		fmt.Printf("Current Quote Reserve: %d\n", pool.QuoteReserve)

		// Calculate Progress
		progress := float64(pool.QuoteReserve) / float64(poolConfig.MigrationQuoteThreshold) * 100
		fmt.Printf("Bonding Curve Progress: %.2f%%\n", progress)

		// Similar to original example: Calculate quote reserve for a specific price
		// Here we use the current SqrtPrice from the pool as an example
		currentSqrtPrice := u128ToBig(pool.SqrtPrice)
		totalAmount, err := math.GetQuoteReserveFromNextSqrtPrice(currentSqrtPrice, poolConfig)
		if err != nil {
			fmt.Printf("Note: Could not calculate theoretical reserve from current price: %v\n", err)
		} else {
			fmt.Printf("Theoretical quote amount for current sqrt_price %s is: %s\n", currentSqrtPrice.String(), totalAmount.String())
		}

	} else {
		log.Fatal("Could not find any Meteora DBC pool where this token is the BaseMint.")
	}
}

// func main() {
// 	GetBondingCurveProgressFromMint()
// }
