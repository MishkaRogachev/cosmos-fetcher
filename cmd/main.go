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

	// 1. Test the RPC client availability by sending a GET request to the RPC URL
	rpcURL := os.Args[1]
	httpClient := &http.Client{}
	resp, err := httpClient.Get(rpcURL)
	if err != nil {
		log.Fatalf("Cannot reach RPC endpoint: %v", err)
	} else if resp.StatusCode != 200 {
		log.Fatalf("Unexpected response status: %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	// 2. Fetch the node status
	rpcClient := protocol.NewRPCClient(rpcURL, httpClient)
	status, err := rpcClient.Status()
	if err != nil {
		log.Fatalf("Error fetching node status: %v", err)
	}

	fmt.Printf("Connected to RPC endpoint url: %s\n", rpcURL)

	// 3. Parse the sync info from the node status to get earliest and latest block heights
	var syncInfo protocol.SyncInfo
	syncInfoData, ok := status["result"].(map[string]interface{})["sync_info"].(map[string]interface{})
	if !ok {
		log.Fatalf("Error getting sync_info from node status")
	}
	if err := protocol.ParseSyncInfo(syncInfoData, &syncInfo); err != nil {
		log.Fatalf("Error parsing sync_info: %v", err)
	}
	fmt.Printf("Block ranges: %v\n", syncInfo)

	// 4. Fetch blocks
	batchBlockFetcher := protocol.NewBatchBlockFetcher(
		rpcClient, syncInfo.EarliestBlockHeight, syncInfo.LatestBlockHeight, 5,
	)
	blockChannel := batchBlockFetcher.StartFetchingBlocks()

	for block := range blockChannel {
		fmt.Printf("Fetched Block Height: %d\n", block.BlockHeight)
	}
}
