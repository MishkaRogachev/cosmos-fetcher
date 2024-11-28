package persistence

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

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

func (bs *BlockStore) writeBlocksToFile(blocks []*protocol.Block) error {
	if len(blocks) == 0 {
		return nil
	}

	startHeight := blocks[0].BlockHeight
	endHeight := blocks[len(blocks)-1].BlockHeight
	filePath := filepath.Join(bs.blockDir, fmt.Sprintf("%d_%d_blocks.json", startHeight, endHeight))

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

func (bs *BlockStore) SaveBlock(block *protocol.Block) error {
	startHeight := (block.BlockHeight / int64(bs.blocksPerFile)) * int64(bs.blocksPerFile)

	// Find the position to insert the block in sorted order
	inserted := false
	for i, b := range bs.blockMap[startHeight] {
		if b.BlockHeight > block.BlockHeight {
			bs.blockMap[startHeight] = append(bs.blockMap[startHeight][:i], append([]*protocol.Block{block}, bs.blockMap[startHeight][i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		bs.blockMap[startHeight] = append(bs.blockMap[startHeight], block)
	}

	block_count := len(bs.blockMap[startHeight])
	if block_count >= bs.blocksPerFile {
		if err := bs.writeBlocksToFile(bs.blockMap[startHeight]); err != nil {
			return err
		}
		// Clear the list after writing to file
		bs.blockMap[startHeight] = nil
	}

	return nil
}

func (bs *BlockStore) Close() {
	for _, blocks := range bs.blockMap {
		if len(blocks) > 0 {
			if err := bs.writeBlocksToFile(blocks); err != nil {
				log.Printf("Error writing blocks to file: %v", err)
			}
		}
	}
}
