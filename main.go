package main

import (
	"flag"
	"fmt"
	"os"
)

const version = "0.1.0"

func main() {
	// Define command-line flags
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help information")

	flag.Parse()

	if *showVersion {
		fmt.Printf("MxM-Chain v%s\n", version)
		fmt.Println("A blockchain implementation in Go")
		os.Exit(0)
	}

	if *showHelp || len(os.Args) == 1 {
		printHelp()
		os.Exit(0)
	}

	// If no recognized flags were provided, show help
	fmt.Println("Unknown command. Use -help for usage information.")
	os.Exit(1)
}

func printHelp() {
	fmt.Printf("MxM-Chain v%s - Blockchain Implementation\n\n", version)
	fmt.Println("Usage:")
	fmt.Println("  mxm-chain [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -version  Show version information")
	fmt.Println("  -help     Show this help message")
	fmt.Println()
	fmt.Println("Available Commands:")
	fmt.Println("  miner     Start the mining node (use: go run cmd/miner/main.go)")
	fmt.Println("  storage   Manage blockchain storage (use: go run cmd/storage/main.go)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Show version")
	fmt.Println("  go run main.go -version")
	fmt.Println()
	fmt.Println("  # Start mining node")
	fmt.Println("  go run cmd/miner/main.go")
	fmt.Println()
	fmt.Println("  # Manage storage")
	fmt.Println("  go run cmd/storage/main.go -help")
	fmt.Println()
	fmt.Println("Documentation:")
	fmt.Println("  See README.md and SPRINTS.md for more information")
}
