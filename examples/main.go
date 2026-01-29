package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	// Shift args so that the called functions see their expected arguments at os.Args[1]
	// Original: [binary, cmd, arg1, arg2...]
	// New:      [binary, arg1, arg2...]
	os.Args = append([]string{os.Args[0]}, os.Args[2:]...)

	switch cmd {
	case "claim_creator_trading_fee":
		ClaimCreatorTradingFee()
	case "claim_partner_trading_fee":
		ClaimPartnerTradingFee()
	case "create_pool_and_swap_sol":
		CreatePoolAndSwapSol()
	case "create_pool_and_swap_usdc":
		CreatePoolAndSwapUsdc()
	case "get_bonding_curve_progress":
		GetBondingCurveProgress()
	case "get_bonding_curve_progress_from_mint":
		GetBondingCurveProgressFromMint()
	case "get_pool":
		GetPool()
	case "get_pool_config":
		GetPoolConfig()
	case "get_pool_config_from_mint":
		GetPoolConfigFromMint()
	case "get_pool_fee_metrics":
		GetPoolFeeMetrics()
	case "monitor_bonding_curve_graduation":
		monitorBondingCurveGraduation()
	case "remove_liquidity":
		RemoveLiquidity()
	case "transfer_pool_creator":
		TransferPoolCreator()
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: go run ./examples <command> [args...]")
	fmt.Println("Available commands:")
	fmt.Println("  claim_creator_trading_fee")
	fmt.Println("  claim_partner_trading_fee")
	fmt.Println("  create_pool_and_swap_sol")
	fmt.Println("  create_pool_and_swap_usdc")
	fmt.Println("  get_bonding_curve_progress")
	fmt.Println("  get_bonding_curve_progress_from_mint")
	fmt.Println("  get_pool")
	fmt.Println("  get_pool_config")
	fmt.Println("  get_pool_config_from_mint")
	fmt.Println("  get_pool_fee_metrics")
	fmt.Println("  monitor_bonding_curve_graduation")
	fmt.Println("  remove_liquidity")
	fmt.Println("  transfer_pool_creator")
}
