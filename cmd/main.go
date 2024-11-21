package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/MishkaRogachev/cosmos-fetcher/protocol"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <cosmos_rpc_url>", os.Args[0])
	}

	rpcURL := os.Args[1]
	httpClient := &http.Client{}
	resp, err := httpClient.Get(rpcURL)
	if err != nil {
		log.Fatalf("Cannot reach RPC endpoint: %v", err)
	} else if resp.StatusCode != 200 {
		log.Fatalf("Unexpected response status: %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	rpcClient := protocol.NewRPCClient(rpcURL, httpClient)
	status, err := rpcClient.Status()
	if err != nil {
		log.Fatalf("Error fetching node status: %v", err)
	}

	fmt.Printf("Connected to RPC endpoint url: %s\n", rpcURL)
	fmt.Printf("Node Status: %v\n", status)

	fmt.Printf("Connected to RPC endpoint url: %s\n", rpcURL)
}
