package main

import (
	"errors"
	"os"
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/kinecosystem/agora-common/kin"
	"github.com/kinecosystem/agora-common/webhook/events"
	"github.com/kinecosystem/kin-go/client"
	"github.com/mr-tron/base58/base58"

	"fmt"
	"log"
)

func main() {
	router := gin.Default()
	// same as
	// config := cors.DefaultConfig()
	// config.AllowAllOrigins = true
	// router.Use(cors.New(config))
	router.Use(cors.Default())

	godotenv.Load()

	fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
	fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
	fmt.Println("Kin Go SDK App")

	var appIndex uint16 = 0
	var appIndexRaw, appIndexError = strconv.ParseInt(os.Getenv("APP_INDEX"), 0, 16)
	if appIndexError != nil {
		appIndexRaw = 0
	} else {
		appIndex = uint16(appIndexRaw)
	}
	fmt.Println("App Index -", appIndex)

	var kinClient client.Client
	var kinClientEnv = client.EnvironmentTest

	var appHotWallet, appHotWalletError = kin.PrivateKeyFromString((os.Getenv("PRIVATE_KEY")))
	if appHotWalletError != nil {
		log.Fatal(appHotWalletError)
	}

	var appTokenAccounts []kin.PublicKey
	var appUserName = "App"
	var appPublicKey = appHotWallet.Public().Base58()
	fmt.Println("App Public Key -", appPublicKey)
	fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
	fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")

	type User struct {
		Name             string
		PublicKey        string
		PrivateKey       kin.PrivateKey
		KinTokenAccounts []kin.PublicKey
	}

	var testUsers []User = make([]User, 0)
	var prodUsers []User = make([]User, 0)

	getAppUser := func() User {
		return User{appUserName, appPublicKey, appHotWallet, appTokenAccounts}
	}

	getUser := func(name string) (*User, error) {
		var rawUsers []User
		// var user User
		if kinClientEnv == client.EnvironmentTest {
			rawUsers = testUsers
		} else {
			rawUsers = prodUsers
		}

		if name == "App" {
			var appUser = getAppUser()
			return &appUser, nil
		}

		for _, v := range rawUsers {
			if name == v.Name {
				return &v, nil
			}
		}

		return nil, errors.New("No User")

	}

	type SanitisedUser struct {
		Name      string `json:"name"`
		PublicKey string `json:"publicKey"`
	}

	getSanitisedUserData := func(user User) SanitisedUser {
		var data = SanitisedUser{user.Name, user.PublicKey}
		return data
	}

	getUsers := func() []SanitisedUser {
		var rawUsers []User
		var users []SanitisedUser

		if kinClientEnv == client.EnvironmentTest {
			rawUsers = testUsers
		} else {
			rawUsers = prodUsers
		}

		users = append(users, getSanitisedUserData(getAppUser()))
		for _, rawUser := range rawUsers {
			users = append(users, getSanitisedUserData(rawUser))
		}

		return users
	}

	saveUser := func(name string, privateKey kin.PrivateKey, kinTokenAccounts []kin.PublicKey) {
		var newUser = User{name, privateKey.Public().Base58(), privateKey, kinTokenAccounts}
		if kinClientEnv == client.EnvironmentTest {
			testUsers = append(testUsers, newUser)
		} else {
			prodUsers = append(prodUsers, newUser)
		}
	}

	var transactions []string = make([]string, 0)

	saveTransaction := func(transactionId string) {
		transactions = append(transactions, transactionId)
	}

	getTransactionType := func(typeString string) kin.TransactionType {
		if typeString == "P2P" {
			return kin.TransactionTypeP2P
		}
		if typeString == "Earn" {
			return kin.TransactionTypeEarn
		}
		if typeString == "Spend" {
			return kin.TransactionTypeSpend
		}
		return kin.TransactionTypeNone
	}

	router.GET("/status", func(c *gin.Context) {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println(" - get /status")

		var env = 1
		var appIndexResponse = 0
		var users = getUsers()

		if kinClient != nil {
			fmt.Println("Kin Client - exists")
			fmt.Println("App Index -", appIndex)
			fmt.Println("Environment -", kinClientEnv)

			for _, v := range users {
				fmt.Println("user: ", v.Name, v.PublicKey)
			}

			for _, v := range transactions {
				fmt.Println("transaction: ", v)
			}
		} else {
			fmt.Println("Kin Client - nil")
		}

		if kinClient != nil {
			appIndexResponse = int(appIndex)
		}

		if kinClientEnv == client.EnvironmentProd {
			env = 0
		}

		c.JSON(200, gin.H{"appIndex": appIndexResponse, "env": env, "users": users, "transactions": transactions})
	})

	router.POST("/setup", func(c *gin.Context) {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		envString := c.Query("env")
		fmt.Println(" - post /setup", envString)

		var env = client.EnvironmentTest
		if envString == "Prod" {
			env = client.EnvironmentProd
		}

		kinClient = nil
		fmt.Println("appIndex", appIndex)

		newClient, newClientError := client.New(env, client.WithAppIndex(appIndex), client.WithMaxRetries(0))
		if newClientError != nil {
			fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
			fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
			fmt.Println(" - ERROR", newClientError)
			c.Status(400)
		}

		tokenAccounts, tokenAccountsError := newClient.ResolveTokenAccounts(c, appHotWallet.Public())
		if tokenAccountsError != nil {
			fmt.Println("Something went wrong getting your app token accounts!", tokenAccountsError)
			c.Status(400)
		}

		appTokenAccounts = tokenAccounts
		kinClient = newClient
		kinClientEnv = env

		c.Status(200)
	})

	router.POST("/account", func(c *gin.Context) {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		name := c.Query("name")
		fmt.Println(" - post /account", name)

		var privateKey, privateKeyError = kin.NewPrivateKey()
		if privateKeyError != nil {
			fmt.Println("Something went wrong making your new private key!", privateKeyError)
			c.Status(400)

		}

		var createAccountError = kinClient.CreateAccount(c, privateKey)
		if createAccountError != nil {
			fmt.Println("Something went wrong creating your account!", createAccountError)
			c.Status(400)
			return
		}

		var kinTokenAccounts, resolveError = kinClient.ResolveTokenAccounts(c, privateKey.Public())
		if resolveError != nil {
			fmt.Println("Something went wrong resolving your token accounts!", resolveError)
			c.Status(400)
			return
		}

		saveUser(name, privateKey, kinTokenAccounts)

		c.Status(200)
		return
	})

	router.GET("/balance", func(c *gin.Context) {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		name := c.Query("user")
		fmt.Println(" - post /balance", name)

		var user, getUserError = getUser(name)
		if getUserError != nil {
			fmt.Println("Something went wrong finding your User!", getUserError)
			c.Status(400)
			return
		}

		var balanceInQuarks, balanceError = kinClient.GetBalance(c, user.PrivateKey.Public())
		if balanceError != nil {
			fmt.Println("Something went wrong finding your Balance!", balanceError)
			c.Status(400)
			return
		}

		var balanceInKin = kin.FromQuarks(balanceInQuarks)
		fmt.Println("balanceInKin", balanceInKin)

		c.String(200, balanceInKin)
		return
	})

	router.POST("/airdrop", func(c *gin.Context) {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		name := c.Query("to")
		amount := c.Query("amount")
		fmt.Println(" - post /airdrop", name, amount)

		var quarks, quarksError = kin.ToQuarks(amount)
		if quarksError != nil {
			fmt.Println("Something went wrong finding your quarks!", quarksError)
			c.Status(400)
			return
		}

		var user, getUserError = getUser(name)
		if getUserError != nil {
			fmt.Println("Something went wrong finding your User!", getUserError)
			c.Status(400)
			return
		}

		var tokenAccount = user.KinTokenAccounts[0]

		var airdrop, airdropError = kinClient.RequestAirdrop(c, tokenAccount, uint64(quarks))
		if airdropError != nil {
			fmt.Println("Something went wrong with your Airdrop!", airdropError)
			c.Status(400)
			return
		}

		var transactionId = base58.Encode(airdrop)
		fmt.Println("transactionId", transactionId)
		saveTransaction(transactionId)

		c.Status(200)
		return
	})

	type SendKinPayload struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Amount string `json:"amount"`
		Type   string `json:"type"`
	}

	router.POST("/send", func(c *gin.Context) {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")

		var payload SendKinPayload
		c.BindJSON(&payload)

		fromName := payload.From
		toName := payload.To
		amount := payload.Amount
		typeString := payload.Type
		fmt.Println(" - post /send", fromName, toName, amount, typeString)

		var quarks, quarksError = kin.ToQuarks(amount)
		if quarksError != nil {
			fmt.Println("Something went wrong finding your quarks!", quarksError)
			c.Status(400)
			return
		}

		var fromUser, fromUserError = getUser(fromName)
		if fromUserError != nil {
			fmt.Println("Something went wrong finding your User!", fromUserError)
			c.Status(400)
			return
		}
		fmt.Println("fromUser", fromUser.PublicKey)

		var toUser, toUserError = getUser(toName)
		if toUserError != nil {
			fmt.Println("Something went wrong finding your User!", toUserError)
			c.Status(400)
			return
		}
		fmt.Println("toUser", toUser.PublicKey)

		var sender = fromUser.PrivateKey
		var destination = toUser.KinTokenAccounts[0]

		var transactionType = getTransactionType(typeString)

		var transaction, transactionError = kinClient.SubmitPayment(c, client.Payment{
			Sender:      sender,
			Destination: destination,
			Type:        transactionType,
			Quarks:      quarks,
		})

		if transactionError != nil {
			fmt.Println("Something went wrong with your Transaction!", transactionError)
			c.Status(400)
			return
		}

		var transactionId = base58.Encode(transaction)
		fmt.Println("transactionId", transactionId)
		saveTransaction(transactionId)

		c.Status(200)
		return
	})

	type BatchPayment struct {
		To     string `json:"to"`
		Amount string `json:"amount"`
	}

	type SendEarnBatchPayload struct {
		From  string         `json:"from"`
		Batch []BatchPayment `json:"batch"`
	}

	getSanitisedBatchEarn := func(payment BatchPayment) (client.Earn, error) {
		var quarks, quarksError = kin.ToQuarks(payment.Amount)
		if quarksError != nil {
			fmt.Println("Something went wrong finding your quarks!", quarksError)
		}

		var toUser, toUserError = getUser(payment.To)
		if toUserError != nil {
			fmt.Println("Something went wrong finding your User!", toUserError)
		}

		var sanitised = client.Earn{toUser.PrivateKey.Public(), quarks, nil}

		return sanitised, nil
	}

	router.POST("/earn_batch", func(c *gin.Context) {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println(" - post /earn_batch")

		var payload SendEarnBatchPayload
		c.BindJSON(&payload)

		fromName := payload.From
		batch := payload.Batch

		fmt.Println(" - fromName", fromName)
		fmt.Println(" - batch", batch)

		var fromUser, fromUserError = getUser(fromName)
		if fromUserError != nil {
			fmt.Println("Something went wrong finding your User!", fromUserError)
			c.Status(400)
			return
		}
		fmt.Println("fromUser", fromUser.PublicKey)
		var sender = fromUser.PrivateKey

		var earns []client.Earn
		for _, v := range batch {
			earn, earnError := getSanitisedBatchEarn((v))
			if earnError != nil {
				fmt.Println("Something went wrong getting your Batch Earn!", earnError)
				c.Status(400)
				return
			}
			earns = append(earns, earn)
		}

		var transaction, transactionError = kinClient.SubmitEarnBatch(c, client.EarnBatch{
			Sender: sender,
			Earns:  earns,
		})

		if transactionError != nil {
			fmt.Println("Something went wrong making your batch payment!", transactionError)
			c.Status(400)
			return
		}

		var transactionId = base58.Encode(transaction.TxID)
		fmt.Println("transactionId", transactionId)
		saveTransaction(transactionId)

		c.Status(200)
		return
	})

	type SanitisedPayment struct {
		Type        kin.TransactionType `json:"type"`
		Quarks      int64               `json:"quarks"`
		Sender      string              `json:"sender"`
		Destination string              `json:"destination"`
		Memo        string              `json:"memo"`
	}

	getSanitisedPayment := func(payment client.ReadOnlyPayment) SanitisedPayment {
		fmt.Println("type", payment.Type)
		fmt.Println("quarks", payment.Quarks)
		fmt.Println("sender", payment.Sender.Base58())
		fmt.Println("destination", payment.Destination.Base58())
		fmt.Println("memo", payment.Memo)

		var sanitised = SanitisedPayment{payment.Type, payment.Quarks, payment.Sender.Base58(), payment.Destination.Base58(), payment.Memo}

		return sanitised
	}

	type TransactionResponse struct {
		txState  client.TransactionState
		payments []SanitisedPayment
	}

	router.GET("/transaction", func(c *gin.Context) {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		transactionId := c.Query("transaction_id")
		fmt.Println(" - post /transaction", transactionId)

		transactionHash, transactionHashError := base58.Decode(transactionId)
		if transactionHashError != nil {
			fmt.Println("Something went wrong decoding your transaction ID!", transactionHashError)
			c.Status(400)
			return
		}

		var transaction, getTransactionError = kinClient.GetTransaction(c, transactionHash)
		if getTransactionError != nil {
			fmt.Println("Something went wrong finding your Transaction!", getTransactionError)
			c.Status(400)
			return
		}

		var sanitisedPayments []SanitisedPayment
		for _, v := range transaction.Payments {
			sanitisedPayments = append(sanitisedPayments, getSanitisedPayment(v))
		}

		fmt.Println("state", transaction.TxState)
		fmt.Println("payments", sanitisedPayments)

		c.JSON(200, gin.H{"txState": transaction.TxState, "payments": sanitisedPayments})

	})

	// Webhooks
	var webhookSecret = os.Getenv("SERVER_WEBHOOK_SECRET")

	eventsHandler := func(events []events.Event) error {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println(" - Event Webhook")

		for _, e := range events {
			if e.TransactionEvent == nil {
				log.Println("received event:", e)
				continue
			}

			var transactionId = base58.Encode(e.TransactionEvent.TxID)
			fmt.Println("received event - transactionId", transactionId)
		}

		return nil
	}

	router.POST("/events", gin.WrapH(client.EventsHandler(webhookSecret, eventsHandler)))

	signTransactionHandler := func(req client.SignTransactionRequest, resp *client.SignTransactionResponse) error {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println(" - Sign Transaction Webhook")
		txID, err := req.TxID()
		if err != nil {
			return err
		}
		var transactionId = base58.Encode(txID)
		fmt.Println("received sign transaction request - transactionId", transactionId)

		// Note: Agora will _not_ forward a rejected transaction to the blockchain,
		//       but it's safer to check that here as well.
		if resp.IsRejected() {
			fmt.Println("transaction rejected:  ", transactionId, len(req.Payments))
			return nil
		}

		fmt.Println("transaction approved: ", transactionId, len(req.Payments))

		// Note: This allows agora to forward the transaction to the blockchain. However,
		// it does not indicate that it will be submitted successfully, or that the transaction
		// will be successful. For example, if sender has insufficient funds.
		//
		// Backends may keep track of the transaction themselves via the req.TxID(), and rely
		// on either the Events handler or polling to get the status.
		return resp.Sign(appHotWallet)
	}
	router.POST("/sign_transaction", gin.WrapH(client.SignTransactionHandler(webhookSecret, signTransactionHandler)))

	var port = ":3001"
	if os.Getenv("PORT") != "" {
		port = ":" + os.Getenv("PORT")
	}
	fmt.Println("port", port)

	router.Run(port)
}
