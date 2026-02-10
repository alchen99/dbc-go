package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/alchen99/dbc-go/instructions"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/mew-sh/dotenv"
)

func GetPool() {
	env, err := dotenv.Read(".env")

	if err != nil {
		log.Fatal(err)
	}

	rpcClient := rpc.New(getRPCURL())

	poolAddressStr := env["POOL_ADDRESS"]

	fmt.Println("Getting pool", poolAddressStr, "...")
	poolAddress := solana.MustPublicKeyFromBase58(poolAddressStr)

	ctx := context.Background()

	pool, err := instructions.GetPool(ctx, poolAddress, rpcClient)
	if err != nil {
		log.Fatalf("Failed to get pool: %v", err)
	}

	// Marshal the pool to JSON
	jsonData, err := json.MarshalIndent(pool, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal pool to JSON: %v", err)
	}

	fmt.Printf("Pool JSON: %s\n", string(jsonData))

}

// func main() {
// 	GetPool()
// }
