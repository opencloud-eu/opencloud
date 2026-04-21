package main

import (
	"fmt"
	"os"
	"time"

	"github.com/opencloud-eu/opencloud/opencloud/pkg/command"
)

func main() {
	if err := command.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	// INFO: this is a hotfix, we need to replace suture and wait for nats to
	//       gracefully shutdown using rungroups
	// 		 see https://github.com/opencloud-eu/opencloud/issues/2282
	if os.Getenv("TEST_SERVER_URL") == "" {
		fmt.Println("Waiting for 30 seconds before exiting...")
		time.Sleep(30 * time.Second)
		fmt.Println("Exiting...")
	}
}
