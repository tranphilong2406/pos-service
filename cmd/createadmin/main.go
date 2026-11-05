// Command createadmin creates an admin user. The API cannot create the first one,
// because creating users already requires an admin token.
//
//	ADMIN_PASSWORD=... go run ./cmd/createadmin -name admin -full-name "Store owner"
package main

import (
	"StoreServer/job"
	"StoreServer/models"
	"flag"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	os.Exit(run())
}

func run() int {
	name := flag.String("name", "", "admin username (required)")
	fullName := flag.String("full-name", "", "display name")
	flag.Parse()

	// Read the password from the environment so it does not end up in shell history.
	password := os.Getenv("ADMIN_PASSWORD")
	if *name == "" || password == "" {
		fmt.Fprintln(os.Stderr, "usage: ADMIN_PASSWORD=... createadmin -name <username> [-full-name <name>]")
		return 2
	}

	user := models.User{UserName: *name, FullName: *fullName, Role: models.AdminRole}
	if res := user.Validate(); res.Code != http.StatusOK {
		fmt.Fprintln(os.Stderr, res.Message)
		return 1
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	user.Password = string(hash)

	job.DBConnect()
	defer job.Disconnect()
	models.InitAuthDB()

	if res := models.AuthDB.Create(user); res.Code != http.StatusCreated {
		fmt.Fprintln(os.Stderr, res.Message)
		return 1
	}
	fmt.Println("Created admin", *name)
	return 0
}
