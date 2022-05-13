package main

import (
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

	var app_user = User{app_user_name, app_public_key, app_hot_wallet, app_token_accounts}

	var test_users []User = make([]User, 0)
	var prod_users []User = make([]User, 0)

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

		users = append(users, get_sanitised_user_data(app_user))
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

		new_client, err := client.New(env, client.WithAppIndex(app_index), client.WithMaxRetries(0))
		if err != nil {
			fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
			fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
			fmt.Println(" - ERROR", err)
			c.Status(400)
		}

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

	router.Run(":3001")
}
