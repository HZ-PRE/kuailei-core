package main

import (
	"os"

	"github.com/HZ-PRE/kuailei-core/cmd"
)

func main() {
	cmd.ParseCli(os.Args[1:])
}
