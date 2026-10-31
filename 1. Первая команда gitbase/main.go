package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}

	switch os.Args[1] {
	case "init":
		if len(os.Args) != 2 {
			usage()
			os.Exit(2)
		}

		_ = root
		fmt.Println("init command")

	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "use: gitbase init")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
