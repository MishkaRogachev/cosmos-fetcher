package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/MishkaRogachev/cosmos-fetcher/protocol"
)

type BlockStore struct {
	blockDir      string
	blocksPerFile int
	blockMap      map[int64][]*protocol.Block
}

func NewBlockStore(blockDir string, blocksPerFile int) *BlockStore {
	return &BlockStore{
		blockDir:      blockDir,
		blockMap:      make(map[int64][]*protocol.Block),
		blocksPerFile: blocksPerFile,
	}
}

func (bs *BlockStore) SaveBlocks(new_blocks []*protocol.Block, endHeight int64) error {
	// Group blocks by startHeight to reduce io operations
	// Unfinished blocks will be written to file when the next block is processed
	for _, block := range new_blocks {
		startHeight := (block.BlockHeight / int64(bs.blocksPerFile)) * int64(bs.blocksPerFile)
		bs.blockMap[startHeight] = append(bs.blockMap[startHeight], block)
	}

	// Write blocks to their respective files
	for startHeight, blocks := range bs.blockMap {
		block_count := len(blocks)
		if block_count >= bs.blocksPerFile || (block_count > 0 && blocks[block_count-1].BlockHeight == endHeight) {
			// Sort blocks before writing to file, should be fast since they are already mostly sorted
			sort.Slice(blocks, func(i, j int) bool {
				return blocks[i].BlockHeight < blocks[j].BlockHeight
			})

			// Generate file path
			endHeight := min(startHeight+int64(bs.blocksPerFile)-1, endHeight)
			fileName := fmt.Sprintf("%d_%d_blocks.json", startHeight, endHeight)
			filePath := filepath.Join(bs.blockDir, fileName)

			fmt.Printf("Writing blocks to file: %s\n", filePath)

			// Write blocks to file
			if err := bs.writeBlocksToFile(filePath, blocks); err != nil {
				return err
			}
			// Clear the list after writing to file
			bs.blockMap[startHeight] = nil
		}
	}

	return nil
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
