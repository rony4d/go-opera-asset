package main

import (
	"fmt"
	"os"

	"github.com/rony4d/go-opera-asset/cmd/opera/launcher"
)

func main() {

	// Gather the full list of command-line arguments
	arguments := os.Args

	// Call intot he launcher and capture any resulting error

	err := launcher.Launch(arguments)

	if err != nil {

		// Report the issue to stderr/stdout so the user sees it
		fmt.Println("Error:", err)

		// Exit with a non-zero status code to indicate failure
		os.Exit(1)
		return
	}

	// Optional: add a success path if you want to confirm everything worked.
	fmt.Println("Asset Chain Opera Node Started Successfully")

}
