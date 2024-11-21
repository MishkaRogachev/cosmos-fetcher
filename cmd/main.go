package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <cosmos_rpc_url>", os.Args[0])
	}

	rpcURL := os.Args[1]

	fmt.Printf("RPC endpoint url: %s\n", rpcURL)
}
