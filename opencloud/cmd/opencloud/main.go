package main

import (
	"fmt"
	"os"

	"github.com/opencloud-eu/opencloud/opencloud/pkg/command"
)

func main() {
	if err := command.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		fmt.Println(7)
		os.Exit(1)
	}
}
