package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	ccf := flag.Bool("ccf", false, "Short for claim_creator_trading_fee")
	cptf := flag.Bool("cptf", false, "Short for claim_partner_trading_fee")
	cps := flag.Bool("cps", false, "Short for create_pool_and_swap_sol")
	cpu := flag.Bool("cpu", false, "Short for create_pool_and_swap_usdc")
	gbcp := flag.Bool("gbcp", false, "Short for get_bonding_curve_progress")
	gbcpf := flag.Bool("gbcpf", false, "Short for get_bonding_curve_progress_from_mint")
	gp := flag.Bool("gp", false, "Short for get_pool")
	gpc := flag.Bool("gpc", false, "Short for get_pool_config")
	gpcf := flag.Bool("gpcf", false, "Short for get_pool_config_from_mint")
	gpfm := flag.Bool("gpfm", false, "Short for get_pool_fee_metrics")
	mbcg := flag.Bool("mbcg", false, "Short for monitor_bonding_curve_graduation")
	rl := flag.Bool("rl", false, "Short for remove_liquidity")
	tpc := flag.Bool("tpc", false, "Short for transfer_pool_creator")

	flag.Usage = printUsage
	flag.Parse()

	var cmd string
	if *ccf {
		cmd = "claim_creator_trading_fee"
	} else if *cptf {
		cmd = "claim_partner_trading_fee"
	} else if *cps {
		cmd = "create_pool_and_swap_sol"
	} else if *cpu {
		cmd = "create_pool_and_swap_usdc"
	} else if *gbcp {
		cmd = "get_bonding_curve_progress"
	} else if *gbcpf {
		cmd = "get_bonding_curve_progress_from_mint"
	} else if *gp {
		cmd = "get_pool"
	} else if *gpc {
		cmd = "get_pool_config"
	} else if *gpcf {
		cmd = "get_pool_config_from_mint"
	} else if *gpfm {
		cmd = "get_pool_fee_metrics"
	} else if *mbcg {
		cmd = "monitor_bonding_curve_graduation"
	} else if *rl {
		cmd = "remove_liquidity"
	} else if *tpc {
		cmd = "transfer_pool_creator"
	} else {
		if flag.NArg() < 1 {
			printUsage()
			os.Exit(1)
		}
		cmd = flag.Arg(0)
		// Shift args so subcommands see their arguments at os.Args[1]
		os.Args = append([]string{os.Args[0]}, flag.Args()[1:]...)
	}

	// If a flag was used, flag.Args() already contains the arguments after the flags.
	// We need to shift them into os.Args for subcommands that use os.Args.
	if *ccf || *cptf || *cps || *cpu || *gbcp || *gbcpf || *gp || *gpc || *gpcf || *gpfm || *mbcg || *rl || *tpc {
		os.Args = append([]string{os.Args[0]}, flag.Args()...)
	}

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
	fmt.Fprintf(os.Stderr, "Usage: go run ./examples [flag] [args...]\n")
	fmt.Fprintf(os.Stderr, "Available flags:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nAvailable positional commands:\n")
	fmt.Fprintf(os.Stderr, "  claim_creator_trading_fee\n")
	fmt.Fprintf(os.Stderr, "  claim_partner_trading_fee\n")
	fmt.Fprintf(os.Stderr, "  create_pool_and_swap_sol\n")
	fmt.Fprintf(os.Stderr, "  create_pool_and_swap_usdc\n")
	fmt.Fprintf(os.Stderr, "  get_bonding_curve_progress\n")
	fmt.Fprintf(os.Stderr, "  get_bonding_curve_progress_from_mint\n")
	fmt.Fprintf(os.Stderr, "  get_pool\n")
	fmt.Fprintf(os.Stderr, "  get_pool_config\n")
	fmt.Fprintf(os.Stderr, "  get_pool_config_from_mint\n")
	fmt.Fprintf(os.Stderr, "  get_pool_fee_metrics\n")
	fmt.Fprintf(os.Stderr, "  monitor_bonding_curve_graduation\n")
	fmt.Fprintf(os.Stderr, "  remove_liquidity\n")
	fmt.Fprintf(os.Stderr, "  transfer_pool_creator\n")
}
