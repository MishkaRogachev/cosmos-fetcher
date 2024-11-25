package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/MishkaRogachev/cosmos-fetcher/protocol"
)

const BLOCKS_PER_FILE = 32

type BlockStore struct {
	blockDir string
	blockMap map[string][]*protocol.Block
}

func NewBlockStore(blockDir string) *BlockStore {
	return &BlockStore{
		blockDir: blockDir,
		blockMap: make(map[string][]*protocol.Block),
	}
}

func (bs *BlockStore) SaveBlocks(blocks []*protocol.Block) error {
	for _, block := range blocks {
		fileName := bs.getFileNameForBlock(block.BlockHeight)
		filePath := filepath.Join(bs.blockDir, fileName)

		bs.blockMap[filePath] = append(bs.blockMap[filePath], block)

		if len(bs.blockMap[filePath]) >= BLOCKS_PER_FILE {
			if err := bs.writeBlocksToFile(filePath, bs.blockMap[filePath]); err != nil {
				return err
			}
			bs.blockMap[filePath] = nil
		}
	}

	return nil
}

// getFileNameForBlock generates a file name based on the block height
func (bs *BlockStore) getFileNameForBlock(blockHeight int64) string {
	startHeight := (blockHeight / BLOCKS_PER_FILE) * BLOCKS_PER_FILE
	endHeight := startHeight + BLOCKS_PER_FILE - 1
	return fmt.Sprintf("%d_%d_blocks.json", startHeight, endHeight)
}

// writeBlocksToFile writes the blocks to a JSON file
func (bs *BlockStore) writeBlocksToFile(filePath string, blocks []*protocol.Block) error {
	// Sort blocks by height before writing
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].BlockHeight < blocks[j].BlockHeight
	})

	// Create the block directory if it doesn't exist
	if err := os.MkdirAll(bs.blockDir, os.ModePerm); err != nil {
		return err
	}

	// Open the file for writing (create or append)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(blocks); err != nil {
		return err
	}

	return nil
}
