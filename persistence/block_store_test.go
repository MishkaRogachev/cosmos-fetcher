package persistence_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/MishkaRogachev/cosmos-fetcher/persistence"
	"github.com/MishkaRogachev/cosmos-fetcher/protocol"
	"github.com/stretchr/testify/assert"
)

func confirmBlocksInFile(t *testing.T, fileName string, blocks []*protocol.Block) bool {
	fileContent, err := os.ReadFile(fileName)
	if err != nil {
		return false
	}

	var savedBlocks []*protocol.Block
	err = json.Unmarshal(fileContent, &savedBlocks)
	if err != nil {
		return false
	}

	return assert.Equal(t, savedBlocks, blocks)
}

func TestBlockStore_SaveBlock(t *testing.T) {
	blocksPerFile := 3
	blockStore := persistence.NewBlockStore("test_blocks", blocksPerFile)
	defer os.RemoveAll("test_blocks")

	// Create sample blocks
	blocks := []*protocol.Block{
		{BlockHeight: 1},
		{BlockHeight: 2},
		{BlockHeight: 3},
		{BlockHeight: 4},
	}

	// Save blocks to the block store
	for _, block := range blocks {
		err := blockStore.SaveBlock(block)
		assert.NoError(t, err)
	}
	blockStore.Close()

	confirmBlocksInFile(t, "test_blocks/1_3_blocks.json", blocks[:3])
	confirmBlocksInFile(t, "test_blocks/4_4_blocks.json", blocks[3:])
}
