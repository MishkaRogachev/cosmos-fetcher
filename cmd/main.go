package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/MishkaRogachev/cosmos-fetcher/persistence"
	"github.com/MishkaRogachev/cosmos-fetcher/protocol"
)

type Config struct {
	NodeURL     string
	StartHeight int64
	EndHeight   int64
	NumWorkers  int
}

// ParseCLI parses the command line arguments and returns the configuration
func ParseCLI() Config {
	var config Config

	flag.StringVar(&config.NodeURL, "node-url", "", "The Cosmos node RPC endpoint URL")
	flag.Int64Var(&config.StartHeight, "start-height", 0, "The start block height to fetch (default: earliest available)")
	flag.Int64Var(&config.EndHeight, "end-height", 0, "The end block height to fetch (default: latest available)")
	flag.IntVar(&config.NumWorkers, "parallelism", 5, "The number of parallel fetchers to use")

	flag.Parse()

	if config.NodeURL == "" {
		flag.Usage()
		panic("node-url is required")
	}

	return config
}

func main() {
	config := ParseCLI()

	// 1. Test the RPC client availability by sending a GET request to the RPC URL
	httpClient := &http.Client{}
	resp, err := httpClient.Get(config.NodeURL)
	if err != nil {
		log.Fatalf("Cannot reach RPC endpoint: %v", err)
	} else if resp.StatusCode != 200 {
		log.Fatalf("Unexpected response status: %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	// 2. Fetch the node status
	rpcClient := protocol.NewRPCClient(config.NodeURL, httpClient)
	status, err := rpcClient.Status()
	if err != nil {
		log.Fatalf("Error fetching node status: %v", err)
	}

	fmt.Printf("Connected to RPC endpoint url: %s\n", config.NodeURL)

	// 3. Parse the sync info from the node status to get earliest and latest block heights
	var syncInfo protocol.SyncInfo
	syncInfoData, ok := status["result"].(map[string]interface{})["sync_info"].(map[string]interface{})
	if !ok {
		log.Fatalf("Error getting sync_info from node status")
	}
	if err := protocol.ParseSyncInfo(syncInfoData, &syncInfo); err != nil {
		log.Fatalf("Error parsing sync_info: %v", err)
	}

	// 4. Set default block range if not provided
	startHeight := config.StartHeight
	endHeight := config.EndHeight
	if startHeight < syncInfo.EarliestBlockHeight {
		if startHeight != 0 {
			fmt.Println("Start height is earlier than the earliest block height, setting to earliest block height")
		}
		startHeight = syncInfo.EarliestBlockHeight
	}
	if endHeight == 0 || endHeight > syncInfo.LatestBlockHeight {
		if endHeight != 0 {
			fmt.Println("End height is later than the latest block height, setting to latest block height")
		}
		endHeight = syncInfo.LatestBlockHeight
	}
	fmt.Printf("Fetching blocks in range %d-%d using %d workers\n", startHeight, endHeight, config.NumWorkers)

	// 5. Fetch & store blocks
	batchBlockFetcher := protocol.NewBatchBlockFetcher(rpcClient, startHeight, endHeight, config.NumWorkers)
	batchBlockFetcher.StartFetchingBlocks()
	defer batchBlockFetcher.StopFetchingBlocks()

	blockStore := persistence.NewBlockStore("blocks.json")
	defer blockStore.Close()

	for batch := range batchBlockFetcher.BatchChannel {
		for _, block := range batch.Blocks {
			fmt.Printf("Fetched Block Height: %d\n", block.BlockHeight)
			if err := blockStore.SaveBlock(block); err != nil {
				log.Printf("Error saving block: %v", err)
			}
		}
	}
}
