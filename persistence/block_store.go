package persistence

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/MishkaRogachev/cosmos-fetcher/protocol"
)

type BlockStore struct {
	blockDir      string
	blocksPerFile int
	blockMap      map[int64][]*protocol.Block
	mu            sync.Mutex
}

func NewBlockStore(blockDir string, blocksPerFile int) *BlockStore {
	return &BlockStore{
		blockDir:      blockDir,
		blockMap:      make(map[int64][]*protocol.Block),
		blocksPerFile: blocksPerFile,
	}
}

func (bs *BlockStore) writeBlocksToFile(blocks []*protocol.Block) error {
	blockCount := len(blocks)
	if blockCount == 0 {
		return nil
	}

	startHeight := blocks[0].BlockHeight
	endHeight := blocks[blockCount-1].BlockHeight
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
	encoder.SetIndent("", "  ")
	if _, err := file.WriteString("["); err != nil { // Start of the JSON array
		return err
	}
	for i, block := range blocks {
		if err := encoder.Encode(block); err != nil {
			return err
		}
		if i < blockCount-1 {
			if _, err := file.WriteString(","); err != nil { // Write a comma and newline
				return err
			}
		}
	}
	if _, err := file.WriteString("]"); err != nil { // End of the JSON array
		return err
	}

	fmt.Println("Wrote", blockCount, "blocks to file:", filePath)

	return nil
}

func (bs *BlockStore) SaveBlock(block *protocol.Block) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	startHeight := (block.BlockHeight / int64(bs.blocksPerFile)) * int64(bs.blocksPerFile)

	// Find the position to insert the block in sorted order
	isInserted := false
	for i, b := range bs.blockMap[startHeight] {
		if b.BlockHeight > block.BlockHeight {
			bs.blockMap[startHeight] = append(
				bs.blockMap[startHeight][:i],
				append([]*protocol.Block{block}, bs.blockMap[startHeight][i:]...)...,
			)
			isInserted = true
			break
		}
	}
	if !isInserted {
		bs.blockMap[startHeight] = append(bs.blockMap[startHeight], block)
	}

	blockCount := len(bs.blockMap[startHeight])
	if blockCount >= bs.blocksPerFile {
		if err := bs.writeBlocksToFile(bs.blockMap[startHeight]); err != nil {
			return err
		}
		// Clear the list after writing to file
		bs.blockMap[startHeight] = []*protocol.Block{}
	}

	return nil
}

func (bs *BlockStore) Close() {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	for _, blocks := range bs.blockMap {
		if len(blocks) > 0 {
			if err := bs.writeBlocksToFile(blocks); err != nil {
				log.Printf("Error writing blocks to file: %v", err)
			}
		}
	}
}
