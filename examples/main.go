package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mew-sh/dotenv"
)

var (
	rpcUrlFlag  = flag.String("rpc", "", "Override RPC URL")
	mainnetFlag = flag.Bool("mainnet", false, "Use mainnet-beta RPC URL")
)

func main() {
	ccf := flag.Bool("ccf", false, "Run claim_creator_trading_fee")
	cptf := flag.Bool("cptf", false, "Run claim_partner_trading_fee")
	cps := flag.Bool("cps", false, "Run create_pool_and_swap_sol")
	cpu := flag.Bool("cpu", false, "Run create_pool_and_swap_usdc")
	gbcp := flag.Bool("gbcp", false, "Run get_bonding_curve_progress POOL_ADDRESS")
	gbcpf := flag.Bool("gbcpf", false, "Run get_bonding_curve_progress_from_mint TOKEN_MINT_ADDRESS")
	gp := flag.Bool("gp", false, "Run get_pool POOL_ADDRESS")
	gpc := flag.Bool("gpc", false, "Run get_pool_config POOL_ADDRESS")
	gpcf := flag.Bool("gpcf", false, "Run get_pool_config_from_mint TOKEN_MINT_ADDRESS")
	gpfm := flag.Bool("gpfm", false, "Run get_pool_fee_metrics POOL_ADDRESS")
	mbcg := flag.Bool("mbcg", false, "Run monitor_bonding_curve_graduation TOKEN_MINT_ADDRESS")
	rl := flag.Bool("rl", false, "Run remove_liquidity POOL_ADDRESS PERCENTAGE")
	tpc := flag.Bool("tpc", false, "Run transfer_pool_creator NEW_CREATOR_PRIVATE_KEY POOL_ADDRESS")

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

func getRPCURL() string {
	if *rpcUrlFlag != "" {
		log.Printf("Using network: %s", *rpcUrlFlag)
		return *rpcUrlFlag
	}
	if *mainnetFlag {
		log.Printf("Using network: MAINNET")
		return "https://api.mainnet-beta.solana.com"
	}
	env, err := dotenv.Read(".env")
	if err != nil {
		log.Printf("Warning: .env file not found, using default devnet RPC")
		return "https://api.devnet.solana.com"
	}
	log.Printf("Using custom network: %s", env["RPC_URL"])
	return env["RPC_URL"]
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: go run ./examples [flag] [args...]\n")
	fmt.Fprintf(os.Stderr, "Available flags:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nAvailable positional commands:\n")
	fmt.Fprintf(os.Stderr, "  NEW_CREATOR_PRIVATE_KEY\n")
	fmt.Fprintf(os.Stderr, "  TOKEN_MINT_ADDRESS\n")
	fmt.Fprintf(os.Stderr, "  POOL_ADDRESS\n")
	fmt.Fprintf(os.Stderr, "  PERCENTAGE\n")
}
