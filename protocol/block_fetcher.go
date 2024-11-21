package protocol

import (
	"encoding/json"
	"fmt"
	"io"
)

type Block struct {
	BlockHeight     int64
	NumTransactions int
	ChainID         string
}

type BlockFetcher struct {
	client *RPCClient
}

func NewBlockFetcher(client *RPCClient) *BlockFetcher {
	return &BlockFetcher{client: client}
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

	// Check if there's an error field in the response
	if result.Error != nil {
		return nil, fmt.Errorf("RPC error: %s", result.Error.Message)
	}

	numTransactions := len(result.Result.Block.Data.Txs)
	chainID := result.Result.Block.Header.ChainID

	return &Block{
		BlockHeight:     height,
		NumTransactions: numTransactions,
		ChainID:         chainID,
	}, nil
}
