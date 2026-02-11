package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/alchen99/dbc-go/instructions"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/mew-sh/dotenv"
)

func GetPoolConfig() {
	env, err := dotenv.Read(".env")

	if err != nil {
		log.Fatal(err)
	}

	rpcClient := rpc.New(getRPCURL())

	// Get POOL_ADDRESS from command line argument, fallback to .env
	configAddressStr := ""
	if len(os.Args) > 1 {
		configAddressStr = os.Args[1]
	} else {
		configAddressStr = env["POOL_ADDRESS"]
	}

	if configAddressStr == "" {
		log.Fatal("POOL_ADDRESS is required as an argument or in .env")
	}

	fmt.Println("Getting pool config for", configAddressStr, "...")
	configAddress := solana.MustPublicKeyFromBase58(configAddressStr)

	ctx := context.Background()

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
// 	GetPoolConfig()
//}
