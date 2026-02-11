package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/mew-sh/dotenv"

	"github.com/alchen99/dbc-go/helpers"
	"github.com/alchen99/dbc-go/instructions"
)

func TransferPoolCreator() {
	env, err := dotenv.Read(".env")

	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	client := rpc.New(getRPCURL())

	// Get NEW_CREATOR_PRIVATE_KEY from command line argument
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run examples/transfer_pool_creator.go <NEW_CREATOR_PRIVATE_KEY> <POOL_ADDRESS>")
	}

	newCreator, err := solana.PublicKeyFromBase58(os.Args[1])
	if err != nil {
		log.Fatalf("invalid pool address: %v", err)
	}

	// 1) load payer and creator PKs
	payerPrivateKey := env["PAYER_PRIVATE_KEY"]
	if payerPrivateKey == "" {
		log.Fatal("PAYER_PRIVATE_KEY not found in .env")
	}
	payer := solana.MustPrivateKeyFromBase58(payerPrivateKey)

	poolCreatorPrivateKey := env["POOL_CREATOR_PRIVATE_KEY"]
	if poolCreatorPrivateKey == "" {
		log.Fatal("POOL_CREATOR_PRIVATE_KEY not found in .env")
	}
	creator := solana.MustPrivateKeyFromBase58(poolCreatorPrivateKey)

	// 2) virtual pool address
	virtualPool, err := solana.PublicKeyFromBase58(os.Args[2])
	if err != nil {
		log.Fatalf("invalid pool address: %v", err)
	}

	// 3) get pool state to get config
	poolState, err := client.GetAccountInfo(ctx, virtualPool)
	if err != nil {
		log.Fatalf("GetAccountInfo: %v", err)
	}
	if poolState == nil || poolState.Value == nil {
		log.Fatalf("Pool not found")
	}

	// 4) derive PDAs
	migrationMetadata := helpers.DeriveDammV1MigrationMetadataPda(virtualPool)

	// 5) build transfer pool creator instruction
	ixTransfer := instructions.TransferPoolCreator(
		virtualPool,
		poolState.Value.Owner, // config
		creator.PublicKey(),
		newCreator,
		migrationMetadata,
	)

	// 6) assemble transaction
	bh, err := client.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		log.Fatalf("GetLatestBlockhash: %v", err)
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{ixTransfer},
		bh.Value.Blockhash,
		solana.TransactionPayer(payer.PublicKey()),
	)
	if err != nil {
		log.Fatalf("NewTransaction: %v", err)
	}

	// 7) sign with payer and creator
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		switch {
		case key.Equals(payer.PublicKey()):
			return &payer
		case key.Equals(creator.PublicKey()):
			return &creator
		default:
			return nil
		}
	})
	if err != nil {
		log.Fatalf("Sign: %v", err)
	}

	// 8) send & confirm
	sig, err := client.SendTransaction(ctx, tx)
	if err != nil {
		log.Fatalf("SendTransaction: %v", err)
	}
	fmt.Printf("Transaction sent: %s\n", sig)

	// wait for confirmation by polling
	for i := 0; i < 30; i++ { // try for 30 secs
		time.Sleep(time.Second)
		resp, err := client.GetTransaction(ctx, sig, &rpc.GetTransactionOpts{
			Commitment: rpc.CommitmentFinalized,
		})
		if err != nil {
			continue
		}
		if resp != nil {
			if resp.Meta != nil && resp.Meta.Err != nil {
				log.Fatalf("Transaction failed: %v", resp.Meta.Err)
			}
			fmt.Printf("Transaction confirmed: %s\n", `https://solscan.io/tx/`+sig.String())
			return
		}
	}
	log.Fatalf("Transaction confirmation timeout")
}

// func main() {
// 	TransferPoolCreator()
// }
