package protocol_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/MishkaRogachev/cosmos-fetcher/protocol"
	"github.com/stretchr/testify/assert"
)

func formatBlockResponse(w io.Writer, height string) {
	fmt.Fprintf(w, `{"result": {"block": {"header": {"height": "%s", "chain_id": "testchain"}, "data": {"txs": []}}}}`, height)
}

func TestBlockFetcher_FetchBlock(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/block" {
			height := r.URL.Query().Get("height")
			if height == "2" {
				http.Error(w, "block not found", http.StatusNotFound)
				return
			}
			formatBlockResponse(w, height)
			return
		}
		http.NotFound(w, r)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	httpClient := &http.Client{}
	client := protocol.NewRPCClient(server.URL, httpClient)
	fetcher := protocol.NewBlockFetcher(client, 1, 5, 2, 3, 500)

	// Test valid block
	block, err := fetcher.FetchBlock(1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), block.BlockHeight)
	assert.Equal(t, "testchain", block.ChainID)

	// Test error block
	block, err = fetcher.FetchBlock(2)
	assert.Error(t, err)
	assert.Nil(t, block)
}

func TestBlockFetcher_FetchBlockWithRetries(t *testing.T) {
	// Setup a mock HTTP server that fails the first two times
	maxAttempts := 3
	attempts := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < maxAttempts {
			http.Error(w, "temporary error", http.StatusInternalServerError)
			return
		}
		height := r.URL.Query().Get("height")
		formatBlockResponse(w, height)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	httpClient := &http.Client{}
	client := protocol.NewRPCClient(server.URL, httpClient)
	fetcher := protocol.NewBlockFetcher(client, 1, 5, 2, maxAttempts, 10)

	// Test fetching a block with retries
	block, err := fetcher.FetchBlockWithRetries(1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), block.BlockHeight)
	assert.Equal(t, "testchain", block.ChainID)
}

func TestBlockFetcher_StartFetchingBlocks(t *testing.T) {
	// Setup a mock HTTP server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		height := r.URL.Query().Get("height")
		formatBlockResponse(w, height)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	httpClient := &http.Client{}
	client := protocol.NewRPCClient(server.URL, httpClient)
	fetcher := protocol.NewBlockFetcher(client, 1, 10, 5, 3, 1)

	// Listen for blocks in a routine
	var fetchedBlocks []*protocol.Block
	doneChan := make(chan struct{})
	go func() {
		for block := range fetcher.GetChannel() {
			fetchedBlocks = append(fetchedBlocks, block)
			fmt.Printf("Received block %d\n", block.BlockHeight)
		}
		close(doneChan) // Close when we are done reading from the channel
	}()

	// Start fetching blocks
	fetcher.StartFetchingBlocks()

	// Wait for all blocks to be fetched
	<-fetcher.WaitDone()
	<-doneChan
	assert.Len(t, fetchedBlocks, 10, "Expected to fetch 10 blocks")

	// We don't expect the blocks to be in order
	sort.Slice(fetchedBlocks, func(i, j int) bool {
		return fetchedBlocks[i].BlockHeight < fetchedBlocks[j].BlockHeight
	})

	for i, block := range fetchedBlocks {
		assert.Equal(t, int64(i+1), block.BlockHeight)
		assert.Equal(t, "testchain", block.ChainID)
	}
}
