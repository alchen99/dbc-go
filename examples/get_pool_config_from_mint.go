package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/alchen99/dbc-go/common"
	"github.com/alchen99/dbc-go/helpers"
	"github.com/alchen99/dbc-go/instructions"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/mew-sh/dotenv"
)

func GetPoolConfigFromMint() {
	env, err := dotenv.Read(".env")
	if err != nil {
		log.Fatal(err)
	}

	rpcClient := rpc.New(env["RPC_URL"])
	tokenMintStr := env["TOKEN_MINT_ADDRESS"]

	if tokenMintStr == "" {
		log.Fatal("TOKEN_MINT_ADDRESS is required in .env")
	}

	fmt.Println("Looking for pool config for token:", tokenMintStr)
	tokenMint := solana.MustPublicKeyFromBase58(tokenMintStr)
	ctx := context.Background()

	var configAddress solana.PublicKey
	found := false

	// Strategy 1: Check if the token is the BaseMint of a Pool
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
		fmt.Printf("Found %d pool(s) where token is BaseMint.\n", len(pools))
		// Use the first one found
		poolAccount := pools[0]
		fmt.Println("Using Pool Address:", poolAccount.Pubkey.String())

		// Deserialize the pool to get the Config address
		poolData := poolAccount.Account.Data.GetBinary()
		pool, err := helpers.DeserializePool(poolData)
		if err != nil {
			log.Fatalf("Failed to deserialize pool: %v", err)
		}
		configAddress = pool.Config
		found = true
	} else {
		// Strategy 2: Check if the token is the QuoteMint of a PoolConfig
		// PoolConfig Discriminator: [26, 108, 14, 123, 116, 230, 129, 43]
		// QuoteMint Offset: 8
		fmt.Println("Token not found as BaseMint in any pool. Checking if it is a QuoteMint...")

		poolConfigDiscriminator := []byte{26, 108, 14, 123, 116, 230, 129, 43}

		configs, err := rpcClient.GetProgramAccountsWithOpts(
			ctx,
			solana.MustPublicKeyFromBase58(common.DbcProgramID),
			&rpc.GetProgramAccountsOpts{
				Filters: []rpc.RPCFilter{
					{
						Memcmp: &rpc.RPCFilterMemcmp{
							Offset: 0,
							Bytes:  poolConfigDiscriminator,
						},
					},
					{
						Memcmp: &rpc.RPCFilterMemcmp{
							Offset: 8,
							Bytes:  tokenMint.Bytes(),
						},
					},
				},
			},
		)

		if err != nil {
			log.Fatalf("Failed to search for pool configs: %v", err)
		}

		if len(configs) > 0 {
			fmt.Printf("Found %d pool config(s) where token is QuoteMint.\n", len(configs))
			// Use the first one found
			configAccount := configs[0]
			configAddress = configAccount.Pubkey
			found = true
		}
	}

	if !found {
		log.Fatal("Could not find any Meteora DBC pool or config for this token mint.")
	}

	fmt.Println("Getting pool config for Config Address:", configAddress.String(), "...")

	poolConfig, err := instructions.GetPoolConfig(ctx, configAddress, rpcClient)
	if err != nil {
		log.Fatalf("Failed to get pool config: %v", err)
	}

	// Marshal the pool config to JSON
	jsonData, err := json.MarshalIndent(poolConfig, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal pool config to JSON: %v", err)
	}

	fmt.Printf("Pool config JSON: %s\n", string(jsonData))

	fmt.Printf("SqrtStartPrice: %s\n", poolConfig.SqrtStartPrice.String())

	fmt.Println("Curve points:")
	for i, point := range poolConfig.Curve {
		fmt.Printf("Curve[%d] Liquidity: %s\n", i, point.Liquidity.String())
		fmt.Printf("Curve[%d] SqrtPrice: %s\n", i, point.SqrtPrice.String())
	}
}

// func main() {
// 	GetPoolConfigFromMint()
// }
