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

	var app_index uint16 = 0
	var app_index_raw, app_index_err = strconv.ParseInt(os.Getenv("APP_INDEX"), 0, 16)
	if app_index_err != nil {
		app_index_raw = 0
	} else {
		app_index = uint16(app_index_raw)
	}
	fmt.Println("App Index -", app_index)

	var kin_client client.Client
	var kin_client_env = client.EnvironmentTest

	var app_hot_wallet, app_hot_wallet_err = kin.PrivateKeyFromString((os.Getenv("PRIVATE_KEY")))
	if app_hot_wallet_err != nil {
		log.Fatal(app_hot_wallet_err)
	}

	var app_token_accounts []kin.PublicKey
	var app_user_name = "App"
	var app_public_key = app_hot_wallet.Public().Base58()
	fmt.Println("App Public Key -", app_public_key)
	fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
	fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")

	type User struct {
		Name             string
		PublicKey        string
		PrivateKey       kin.PrivateKey
		KinTokenAccounts []kin.PublicKey
	}

	var test_users []User = make([]User, 0)
	var prod_users []User = make([]User, 0)

	get_app_user := func() User {
		return User{app_user_name, app_public_key, app_hot_wallet, app_token_accounts}
	}

	get_user := func(name string) (*User, error) {
		var raw_users []User
		// var user User
		if kin_client_env == client.EnvironmentTest {
			raw_users = test_users
		} else {
			raw_users = prod_users
		}

		if name == "App" {
			var app_user = get_app_user()
			return &app_user, nil
		}

		for _, v := range raw_users {
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

	get_sanitised_user_data := func(user User) SanitisedUser {
		var data = SanitisedUser{user.Name, user.PublicKey}
		return data
	}

	get_users := func() []SanitisedUser {
		var raw_users []User
		var users []SanitisedUser

		if kin_client_env == client.EnvironmentTest {
			raw_users = test_users
		} else {
			raw_users = prod_users
		}

		users = append(users, get_sanitised_user_data(get_app_user()))
		for _, raw_user := range raw_users {
			users = append(users, get_sanitised_user_data(raw_user))
		}

		return users
	}

	save_user := func(name string, private_key kin.PrivateKey, kin_token_accounts []kin.PublicKey) {
		var new_user = User{name, private_key.Public().Base58(), private_key, kin_token_accounts}
		if kin_client_env == client.EnvironmentTest {
			test_users = append(test_users, new_user)
		} else {
			prod_users = append(prod_users, new_user)
		}
	}

	var transactions []string = make([]string, 0)

	save_transaction := func(transaction_id string) {
		transactions = append(transactions, transaction_id)
	}

	get_transaction_type := func(type_string string) kin.TransactionType {
		if type_string == "P2P" {
			return kin.TransactionTypeP2P
		}
		if type_string == "Earn" {
			return kin.TransactionTypeEarn
		}
		if type_string == "Spend" {
			return kin.TransactionTypeSpend
		}
		return kin.TransactionTypeNone
	}

	router.GET("/status", func(c *gin.Context) {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println(" - get /status")

		var env = 1
		var appIndex = 0
		var users = get_users()

		if kin_client != nil {
			fmt.Println("Kin Client - exists")
			fmt.Println("App Index -", app_index)
			fmt.Println("Environment -", kin_client_env)

			for _, v := range users {
				fmt.Println("user: ", v.Name, v.PublicKey)
			}

			for _, v := range transactions {
				fmt.Println("transaction: ", v)
			}
		} else {
			fmt.Println("Kin Client - nil")
		}

		if kin_client != nil {
			appIndex = int(app_index)
		}

		if kin_client_env == client.EnvironmentProd {
			env = 0
		}

		c.JSON(200, gin.H{"appIndex": appIndex, "env": env, "users": users, "transactions": transactions})
	})

	router.POST("/setup", func(c *gin.Context) {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		env_string := c.Query("env")
		fmt.Println(" - post /setup", env_string)

		var env = client.EnvironmentTest
		if env_string == "Prod" {
			env = client.EnvironmentProd
		}

		kin_client = nil
		fmt.Println("app_index", app_index)

		new_client, new_client_error := client.New(env, client.WithAppIndex(app_index), client.WithMaxRetries(0))
		if new_client_error != nil {
			fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
			fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
			fmt.Println(" - ERROR", new_client_error)
			c.Status(400)
		}

		token_accounts, token_accounts_error := new_client.ResolveTokenAccounts(c, app_hot_wallet.Public())
		if token_accounts_error != nil {
			fmt.Println("Something went wrong getting your app token accounts!", token_accounts_error)
			c.Status(400)
		}

		app_token_accounts = token_accounts
		kin_client = new_client
		kin_client_env = env

		c.Status(200)
	})

	router.POST("/account", func(c *gin.Context) {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		name := c.Query("name")
		fmt.Println(" - post /account", name)

		var private_key, private_key_error = kin.NewPrivateKey()
		if private_key_error != nil {
			fmt.Println("Something went wrong making your new private key!", private_key_error)
			c.Status(400)

		}

		var create_account_error = kin_client.CreateAccount(c, private_key)
		if create_account_error != nil {
			fmt.Println("Something went wrong creating your account!", create_account_error)
			c.Status(400)
			return
		}

		var kin_token_accounts, resolve_error = kin_client.ResolveTokenAccounts(c, private_key.Public())
		if resolve_error != nil {
			fmt.Println("Something went wrong resolving your token accounts!", resolve_error)
			c.Status(400)
			return
		}

		save_user(name, private_key, kin_token_accounts)

		c.Status(200)
		return
	})

	router.GET("/balance", func(c *gin.Context) {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		name := c.Query("user")
		fmt.Println(" - post /balance", name)

		var user, get_user_error = get_user(name)
		if get_user_error != nil {
			fmt.Println("Something went wrong finding your User!", get_user_error)
			c.Status(400)
			return
		}

		var balance_in_quarks, balance_error = kin_client.GetBalance(c, user.PrivateKey.Public())
		if balance_error != nil {
			fmt.Println("Something went wrong finding your Balance!", balance_error)
			c.Status(400)
			return
		}

		var balance_in_kin = kin.FromQuarks(balance_in_quarks)
		fmt.Println("balance_in_kin", balance_in_kin)

		c.String(200, balance_in_kin)
		return
	})

	router.POST("/airdrop", func(c *gin.Context) {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		name := c.Query("to")
		amount := c.Query("amount")
		fmt.Println(" - post /airdrop", name, amount)

		// var amount_64, amount_64_error = strconv.ParseInt(amount, 0, 64)
		// fmt.Println("amount_64_error", amount_64_error)
		var quarks, quarks_error = kin.ToQuarks(amount)
		if quarks_error != nil {
			fmt.Println("Something went wrong finding your quarks!", quarks_error)
			c.Status(400)
			return
		}

		var user, get_user_error = get_user(name)
		if get_user_error != nil {
			fmt.Println("Something went wrong finding your User!", get_user_error)
			c.Status(400)
			return
		}

		var token_account = user.KinTokenAccounts[0]

		var airdrop, airdrop_error = kin_client.RequestAirdrop(c, token_account, uint64(quarks))
		if airdrop_error != nil {
			fmt.Println("Something went wrong with your Airdrop!", airdrop_error)
			c.Status(400)
			return
		}

		var transaction_id = base58.Encode(airdrop)
		fmt.Println("transaction_id", transaction_id)
		save_transaction(transaction_id)

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

		from_name := payload.From
		to_name := payload.To
		amount := payload.Amount
		type_string := payload.Type
		fmt.Println(" - post /send", from_name, to_name, amount, type_string)

		var quarks, quarks_error = kin.ToQuarks(amount)
		if quarks_error != nil {
			fmt.Println("Something went wrong finding your quarks!", quarks_error)
			c.Status(400)
			return
		}

		var from_user, from_user_error = get_user(from_name)
		if from_user_error != nil {
			fmt.Println("Something went wrong finding your User!", from_user_error)
			c.Status(400)
			return
		}
		fmt.Println("from_user", from_user.PublicKey)

		var to_user, to_user_error = get_user(to_name)
		if to_user_error != nil {
			fmt.Println("Something went wrong finding your User!", to_user_error)
			c.Status(400)
			return
		}
		fmt.Println("to_user", to_user.PublicKey)

		var sender = from_user.PrivateKey
		var destination = to_user.KinTokenAccounts[0]

		var transaction_type = get_transaction_type(type_string)

		var transaction, transaction_error = kin_client.SubmitPayment(c, client.Payment{
			Sender:      sender,
			Destination: destination,
			Type:        transaction_type,
			Quarks:      quarks,
		})

		if transaction_error != nil {
			fmt.Println("Something went wrong with your Transaction!", transaction_error)
			c.Status(400)
			return
		}

		var transaction_id = base58.Encode(transaction)
		fmt.Println("transaction_id", transaction_id)
		save_transaction(transaction_id)

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

	get_sanitised_payment := func(payment client.ReadOnlyPayment) SanitisedPayment {
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
		transaction_id := c.Query("transaction_id")
		fmt.Println(" - post /transaction", transaction_id)

		transaction_hash, transaction_hash_error := base58.Decode(transaction_id)
		if transaction_hash_error != nil {
			fmt.Println("Something went wrong decoding your transaction ID!", transaction_hash_error)
			c.Status(400)
			return
		}

		var transaction, get_transaction_error = kin_client.GetTransaction(c, transaction_hash)
		if get_transaction_error != nil {
			fmt.Println("Something went wrong finding your Transaction!", get_transaction_error)
			c.Status(400)
			return
		}

		var sanitised_payments []SanitisedPayment
		for _, v := range transaction.Payments {
			sanitised_payments = append(sanitised_payments, get_sanitised_payment(v))
		}

		fmt.Println("state", transaction.TxState)
		fmt.Println("payments", sanitised_payments)

		c.JSON(200, gin.H{"txState": transaction.TxState, "payments": sanitised_payments})

	})

	// Webhooks
	var webhook_secret = os.Getenv("SERVER_WEBHOOK_SECRET")

	eventsHandler := func(events []events.Event) error {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println(" - Event Webhook")

		for _, e := range events {
			if e.TransactionEvent == nil {
				log.Println("received event:", e)
				continue
			}

			var transaction_id = base58.Encode(e.TransactionEvent.TxID)
			fmt.Println("received event - transaction_id", transaction_id)
		}

		return nil
	}

	router.POST("/events", gin.WrapH(client.EventsHandler(webhook_secret, eventsHandler)))

	signTransactionHandler := func(req client.SignTransactionRequest, resp *client.SignTransactionResponse) error {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println(" - Sign Transaction Webhook")
		txID, err := req.TxID()
		if err != nil {
			return err
		}
		var transaction_id = base58.Encode(txID)
		fmt.Println("received sign transaction request - transaction_id", transaction_id)

		// Note: Agora will _not_ forward a rejected transaction to the blockchain,
		//       but it's safer to check that here as well.
		if resp.IsRejected() {
			fmt.Println("transaction rejected:  ", transaction_id, len(req.Payments))
			return nil
		}

		fmt.Println("transaction approved: ", transaction_id, len(req.Payments))

		// Note: This allows agora to forward the transaction to the blockchain. However,
		// it does not indicate that it will be submitted successfully, or that the transaction
		// will be successful. For example, if sender has insufficient funds.
		//
		// Backends may keep track of the transaction themselves via the req.TxID(), and rely
		// on either the Events handler or polling to get the status.
		return resp.Sign(app_hot_wallet)
	}
	router.POST("/sign_transaction", gin.WrapH(client.SignTransactionHandler(webhook_secret, signTransactionHandler)))

	var port = ":3001"
	if os.Getenv("PORT") != "" {
		port = ":" + os.Getenv("PORT")
	}
	fmt.Println("port", port)

	router.Run(port)
}
