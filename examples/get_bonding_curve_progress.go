package main

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"github.com/alchen99/dbc-go/instructions"
	"github.com/alchen99/dbc-go/math"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func GetBondingCurveProgress() {
	rpcClient := rpc.New("https://api.mainnet-beta.solana.com")

	configAddressStr := "YOUR_CONFIG_ADDRESS"
	configAddress := solana.MustPublicKeyFromBase58(configAddressStr)

	ctx := context.Background()

	poolConfig, err := instructions.GetPoolConfig(ctx, configAddress, rpcClient)
	if err != nil {
		log.Fatalf("Failed to get pool config: %v", err)
	}

	fmt.Printf("Migration Quote Threshold: %+v\n", poolConfig.MigrationQuoteThreshold)

	nextSqrtPriceStr := "NEXT_SQRT_PRICE"
	nextSqrtPrice, ok := new(big.Int).SetString(nextSqrtPriceStr, 10)
	if !ok {
		log.Fatalf("Failed to parse next_sqrt_price")
	}

	totalAmount, err := math.GetQuoteReserveFromNextSqrtPrice(nextSqrtPrice, poolConfig)
	if err != nil {
		log.Fatalf("Failed to get quote token from sqrt price: %v", err)
	}

	fmt.Printf("Total quote amount for sqrt_price %s is: %s\n", nextSqrtPrice.String(), totalAmount.String())
}

func main() {
	GetBondingCurveProgress()
}
