package protocol

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"
)

type Block struct {
	BlockHeight     int64
	NumTransactions int
	ChainID         string
}

type BlockFetcher struct {
	client           *RPCClient
	startBlockHeight int64
	endBlockHeight   int64
	numWorkers       int
	maxRetries       int
	retryInterval    int
	channel          chan *Block
	wg               sync.WaitGroup
	done             chan struct{}
	quit             chan struct{}
}

func NewBlockFetcher(client *RPCClient, startBlockHeight, endBlockHeight int64, numWorkers, maxRetries, retryInterval int) *BlockFetcher {
	return &BlockFetcher{
		client:           client,
		startBlockHeight: startBlockHeight,
		endBlockHeight:   endBlockHeight,
		numWorkers:       numWorkers,
		maxRetries:       maxRetries,
		retryInterval:    retryInterval,
		channel:          make(chan *Block),
		quit:             make(chan struct{}),
		done:             make(chan struct{}),
	}
}

func (bf *BlockFetcher) FetchBlock(height int64) (*Block, error) {
	url := fmt.Sprintf("%s/block?height=%d", bf.client.RPCURL, height)
	resp, err := bf.client.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Result BlockResult `json:"result"`
		Error  *RPCError   `json:"error"`
	}
	if result.Error != nil {
		return nil, fmt.Errorf("RPC error: %s", result.Error.Message)
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	numTransactions := len(result.Result.Block.Data.Txs)
	chainID := result.Result.Block.Header.ChainID

	blockHeight, err := strconv.ParseInt(result.Result.Block.Header.Height, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse block height: %v", err)
	}

	if blockHeight != height {
		return nil, fmt.Errorf("unexpected block height: %d", blockHeight)
	}

	return &Block{
		BlockHeight:     blockHeight,
		NumTransactions: numTransactions,
		ChainID:         chainID,
	}, nil
}

func (bf *BlockFetcher) FetchBlockWithRetries(height int64) (*Block, error) {
	for i := 0; i < bf.maxRetries; i++ {
		block, err := bf.FetchBlock(height)
		if err == nil {
			return block, nil
		}

		fmt.Printf("Error fetching block at height %d\n", height)

		if i < bf.maxRetries-1 {
			fmt.Printf("Retrying in %d milliseconds...\n", bf.retryInterval)
			time.Sleep(time.Duration(bf.retryInterval) * time.Millisecond)
		}
	}

	return nil, fmt.Errorf("failed to fetch block at height %d after %d retries", height, bf.maxRetries)
}

func (bf *BlockFetcher) fetchBlocksForWorkerID(workerID int) {
	// Notify that this worker is done when the function returns
	defer bf.wg.Done()

	fmt.Println("Worker", workerID, "starts..")

	for cursor := bf.startBlockHeight + int64(workerID); cursor <= bf.endBlockHeight; cursor += int64(bf.numWorkers) {
		select {
		case <-bf.quit:
			return
		default:
			block, err := bf.FetchBlockWithRetries(cursor)
			if err != nil {
				// Log the error but continue with the next blocks
				fmt.Printf("Skip block %d because of error: %v\n", cursor, err)
				continue
			}

			bf.channel <- block
		}
	}

	fmt.Println("Worker", workerID, "finished")
}

func (bf *BlockFetcher) StartFetchingBlocks() {
	for i := 0; i < bf.numWorkers; i++ {
		bf.wg.Add(1)
		go bf.fetchBlocksForWorkerID(i)
	}

	// Wait for all workers to complete, then close the channel and signal done
	go func() {
		println("Waiting for workers to finish..")
		bf.wg.Wait()
		println("All workers finished")
		close(bf.channel)
		close(bf.done)
	}()
}

func (bf *BlockFetcher) StopFetchingBlocks() {
	close(bf.quit)
	bf.wg.Wait()
}

func (bf *BlockFetcher) WaitDone() <-chan struct{} {
	return bf.done
}

func (bf *BlockFetcher) GetChannel() <-chan *Block {
	return bf.channel
}
