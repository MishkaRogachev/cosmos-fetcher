package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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

		if bs.blockMap[filePath] == nil {
			bs.blockMap[filePath] = []*protocol.Block{}
		}

		// Insert block into the appropriate position according to bklock height
		index := len(bs.blockMap[filePath])
		for i, b := range bs.blockMap[filePath] {
			if block.BlockHeight < b.BlockHeight {
				index = i
				break
			}
		}
		bs.blockMap[filePath] = append(bs.blockMap[filePath], nil)           // Increase the slice size by 1
		copy(bs.blockMap[filePath][index+1:], bs.blockMap[filePath][index:]) // Shift elements to the right
		bs.blockMap[filePath][index] = block

		if len(bs.blockMap[filePath]) >= BLOCKS_PER_FILE {
			if err := bs.writeBlocksToFile(filePath, bs.blockMap[filePath]); err != nil {
				return err
			}
			bs.blockMap[filePath] = nil
		}
	}

	return nil
}

func (bs *BlockStore) getFileNameForBlock(blockHeight int64) string {
	startHeight := (blockHeight / BLOCKS_PER_FILE) * BLOCKS_PER_FILE
	endHeight := startHeight + BLOCKS_PER_FILE - 1
	return fmt.Sprintf("%d_%d_blocks.json", startHeight, endHeight)
}

func (bs *BlockStore) writeBlocksToFile(filePath string, blocks []*protocol.Block) error {
	if err := os.MkdirAll(bs.blockDir, os.ModePerm); err != nil {
		return err
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
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
