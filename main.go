package main

import (
	"errors"
	"os"
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/kinecosystem/agora-common/kin"
	"github.com/kinecosystem/kin-go/client"

	"github.com/joho/godotenv"

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
		fmt.Println("get_user", name)
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

		for i, v := range raw_users {
			fmt.Println("loop i", i)
			fmt.Println("loop v", v)
			fmt.Println("loop v", v.Name)

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
		for i, raw_user := range raw_users {
			fmt.Println(i, raw_user)
			users = append(users, get_sanitised_user_data(raw_user))
		}

		return users
	}

	save_user := func(name string, private_key kin.PrivateKey, kin_token_accounts []kin.PublicKey) {
		fmt.Println("save_user")
		fmt.Println(name)
		fmt.Println(private_key.Public().Base58())
		fmt.Println(kin_token_accounts)

		var new_user = User{name, private_key.Public().Base58(), private_key, kin_token_accounts}

		if kin_client_env == client.EnvironmentTest {
			test_users = append(test_users, new_user)
		} else {
			prod_users = append(prod_users, new_user)
		}
	}

	var transactions []string = make([]string, 0)

	router.GET("/status", func(c *gin.Context) {
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println(" - get /status")

		var users = get_users()
		fmt.Println("users", users)

		var env = 1
		var appIndex = 0

		fmt.Println("kin_client", kin_client)
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
		fmt.Println("private_key", private_key)
		if private_key_error != nil {
			fmt.Println("Something went wrong making your new private key!", private_key_error)
			c.Status(400)

		}

		var create_account_error = kin_client.CreateAccount(c, private_key)
		fmt.Println("create_account_error", create_account_error)
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

		fmt.Println("kin_token_accounts", kin_token_accounts)
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

		fmt.Println("quarks", quarks)

		var user, get_user_error = get_user(name)
		if get_user_error != nil {
			fmt.Println("Something went wrong finding your User!", get_user_error)
			c.Status(400)
			return
		}
		fmt.Println("user", user.PublicKey)
		fmt.Println("tokenAccounts", user.KinTokenAccounts)

		var token_account = user.KinTokenAccounts[0]
		fmt.Println("token_account", token_account)

		var airdrop, airdrop_error = kin_client.RequestAirdrop(c, token_account, uint64(quarks))
		if airdrop_error != nil {
			fmt.Println("Something went wrong with your Airdrop!", airdrop_error)
			c.Status(400)
			return
		}

		fmt.Println("airdrop", airdrop)

		c.Status(200)
		return
	})

	router.Run(":3001")
}
