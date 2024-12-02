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
	NodeURL       string
	StartHeight   int64
	EndHeight     int64
	NumWorkers    int
	BlocksPerFile int
	MaxRetries    int
	RetryInterval int
	ListRanges    bool
}

// ParseCLI parses the command line arguments and returns the configuration
func ParseCLI() Config {
	var config Config

	flag.StringVar(&config.NodeURL, "node-url", "", "The Cosmos node RPC endpoint URL")
	flag.Int64Var(&config.StartHeight, "start-height", 0, "The start block height to fetch (default: earliest available)")
	flag.Int64Var(&config.EndHeight, "end-height", 0, "The end block height to fetch (default: latest available)")
	flag.IntVar(&config.NumWorkers, "parallelism", 5, "The number of parallel fetchers to use")
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

	fmt.Printf("Connected to RPC endpoint url: %s\n", config.NodeURL)

	// 2. Fetch the node status to get earliest and latest block heights
	rpcClient := protocol.NewRPCClient(config.NodeURL, httpClient)
	syncInfo, err := rpcClient.SyncInfo()
	if err != nil {
		log.Fatalf("Error fetching node status: %v", err)
	}

	if config.ListRanges {
		fmt.Printf("Available block ranges: %d-%d\n", syncInfo.EarliestBlockHeight, syncInfo.LatestBlockHeight)
		return
	}

	// 4. Set default block range if not provided
	startHeight := config.StartHeight
	endHeight := config.EndHeight

	if startHeight < syncInfo.EarliestBlockHeight {
		if startHeight != 0 {
			fmt.Printf("Start height is earlier than the earliest block height, setting to earliest block height (%d)\n", syncInfo.EarliestBlockHeight)
		}
		startHeight = syncInfo.EarliestBlockHeight
	}

	if startHeight > endHeight {
		log.Fatalf("Invalid block range: start height is later than end height")
		return
	}
	if startHeight >= syncInfo.LatestBlockHeight {
		log.Fatalf("Start height is later than the latest block height (%d)", syncInfo.LatestBlockHeight)
		return
	}

	if endHeight == 0 || endHeight > syncInfo.LatestBlockHeight {
		if endHeight != 0 {
			fmt.Printf("End height is later than the latest block height, setting to latest block height (%d)\n", syncInfo.LatestBlockHeight)
		}
		endHeight = syncInfo.LatestBlockHeight
	}

	fmt.Printf("Fetching blocks in range %d-%d using %d workers\n", startHeight, endHeight, config.NumWorkers)

	// 5. Fetch & store blocks
	fetcher := protocol.NewBlockFetcher(
		rpcClient,
		startHeight,
		endHeight,
		config.NumWorkers,
		config.MaxRetries,
		config.RetryInterval,
	)

	// Handle graceful shutdown using channel for listening to quit signals
	go func() {
		<-quit
		fmt.Println("\nShutting down gracefully...")
		fetcher.StopFetchingBlocks() // Signal all fetchers to stop
	}()

	fetcher.StartFetchingBlocks()

	blockStore := persistence.NewBlockStore("blocks", config.BlocksPerFile)
	for block := range fetcher.GetChannel() {
		if err := blockStore.SaveBlock(block); err != nil {
			log.Printf("Error saving block: %v", err)
		}
	}

	// Wait for all workers to complete
	<-fetcher.WaitDone()
	blockStore.Close()

	fmt.Println("Exiting!")
}
