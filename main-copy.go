package main

import (
	"context"
	"fmt"
	"log"

	"github.com/kinecosystem/agora-common/kin"
	"github.com/kinecosystem/kin-go/client"
)

func main() {

	sender, err := kin.PrivateKeyFromString("SBASXIKJ2FPKGOXBE4DJL34BIDVL4FNR3ZEZ2ZFNVUYHBVKRAJ5NIKMW")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(sender.Public().Base58())
	dest, err := kin.PublicKeyFromString("CvHoY9hk8LbUhtqmr1rnh2gEUpsfUrQFvJdmGfFg4e5H")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dest.Base58())

	kinClient, err := client.New(client.EnvironmentProd, client.WithAppIndex(360), client.WithMaxRetries(0))
	if err != nil {
		log.Fatal(err)
	}

	// Payment with no invoicing.
	txHash, err := kinClient.SubmitPayment(context.Background(), client.Payment{
		Sender:      sender,
		Destination: dest,
		Type:        kin.TransactionTypeEarn,
		Quarks:      kin.MustToQuarks("2"),
	}, client.WithAccountResolution(client.AccountResolutionPreferred))
	fmt.Printf("Hash: %x, err: %v\n", txHash, err)

}
