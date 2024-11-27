package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/MishkaRogachev/cosmos-fetcher/persistence"
	"github.com/MishkaRogachev/cosmos-fetcher/protocol"
)

type Config struct {
	NodeURL        string
	StartHeight    int64
	EndHeight      int64
	NumWorkers     int
	BlocksPerBatch int
	BlocksPerFile  int
	MaxRetries     int
	RetryInterval  int
	ListRanges     bool
}

// ParseCLI parses the command line arguments and returns the configuration
func ParseCLI() Config {
	var config Config

	flag.StringVar(&config.NodeURL, "node-url", "", "The Cosmos node RPC endpoint URL")
	flag.Int64Var(&config.StartHeight, "start-height", 0, "The start block height to fetch (default: earliest available)")
	flag.Int64Var(&config.EndHeight, "end-height", 0, "The end block height to fetch (default: latest available)")
	flag.IntVar(&config.NumWorkers, "parallelism", 5, "The number of parallel fetchers to use")
	flag.IntVar(&config.BlocksPerBatch, "batch-blocks", 8, "The number of blocks to fetch per batch")
	flag.IntVar(&config.BlocksPerFile, "file-blocks", 16, "The number of blocks to store per file")
	flag.IntVar(&config.MaxRetries, "max-retries", 3, "The maximum number of retries for fetching a block")
	flag.IntVar(&config.RetryInterval, "retry-interval", 500, "The interval in milliseconds between retries")
	flag.BoolVar(&config.ListRanges, "list-ranges", false, "List the available block ranges and exit")

	flag.Parse()

	if config.NodeURL == "" {
		flag.Usage()
		panic("node-url is required")
	}

	return config
}

func main() {
	config := ParseCLI()

	// Signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	// 1. Test the RPC client availability by sending a GET request to the RPC URL
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

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

	if config.ListRanges {
		fmt.Printf("Available block ranges: %d-%d\n", syncInfo.EarliestBlockHeight, syncInfo.LatestBlockHeight)
		return
	}

	// 4. Set default block range if not provided
	startHeight := config.StartHeight
	endHeight := config.EndHeight
	if startHeight > endHeight {
		log.Fatalf("Invalid block range: start height is later than end height")
		return
	}
	if startHeight >= syncInfo.LatestBlockHeight {
		log.Fatalf("Start height is later than the latest block height (%d)", syncInfo.LatestBlockHeight)
		return
	}

	if startHeight < syncInfo.EarliestBlockHeight {
		if startHeight != 0 {
			fmt.Printf("Start height is earlier than the earliest block height, setting to earliest block height (%d)", syncInfo.EarliestBlockHeight)
		}
		startHeight = syncInfo.EarliestBlockHeight
	}
	if endHeight == 0 || endHeight > syncInfo.LatestBlockHeight {
		if endHeight != 0 {
			fmt.Printf("End height is later than the latest block height, setting to latest block height (%d)\n", syncInfo.LatestBlockHeight)
		}
		endHeight = syncInfo.LatestBlockHeight
	}

	fmt.Printf("Fetching blocks in range %d-%d using %d workers\n", startHeight, endHeight, config.NumWorkers)

	// 5. Fetch & store blocks
	batchBlockFetcher := protocol.NewBatchBlockFetcher(
		rpcClient,
		startHeight,
		endHeight,
		config.NumWorkers,
		config.BlocksPerBatch,
		config.MaxRetries,
		config.RetryInterval,
	)
	blockStore := persistence.NewBlockStore("blocks", config.BlocksPerFile)

	batchBlockFetcher.StartFetchingBlocks()

	// Handle graceful shutdown using channel for listening to quit signals
	go func() {
		<-quit
		fmt.Println("\nShutting down gracefully...")
		batchBlockFetcher.StopFetchingBlocks() // Signal all fetchers to stop
	}()

	for batch := range batchBlockFetcher.BatchChannel {
		fmt.Printf("Fetched blocks: %d - %d\n", batch.StartBlockHeight, batch.EndBlockHeight)

		if err := blockStore.SaveBlocks(batch.Blocks, endHeight); err != nil {
			log.Printf("Error saving blocks: %v", err)
		}
	}

	// Wait for all workers to complete
	<-batchBlockFetcher.WaitDone()

	fmt.Println("Exiting.")
}
