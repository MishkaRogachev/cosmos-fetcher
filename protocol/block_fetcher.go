package protocol

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"
)

type Block struct {
	BlockHeight     int64
	NumTransactions int
	ChainID         string
}

type BlockFetcher struct {
	client        *RPCClient
	maxRetries    int
	retryInterval int
}

func NewBlockFetcher(client *RPCClient, maxRetries, retryInterval int) *BlockFetcher {
	return &BlockFetcher{
		client:        client,
		maxRetries:    maxRetries,
		retryInterval: retryInterval,
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
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, fmt.Errorf("RPC error: %s", result.Error.Message)
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

		fmt.Printf("Error fetching block at height %d: %v\n", height, err)

		if i < bf.maxRetries-1 {
			fmt.Printf("Retrying in %d seconds...\n", bf.retryInterval)
			fmt.Printf("Retrying in %d milliseconds...\n", bf.retryInterval)
			time.Sleep(time.Duration(bf.retryInterval) * time.Millisecond)
		}
	}

	return nil, fmt.Errorf("failed to fetch block at height %d after %d retries", height, bf.maxRetries)
}
