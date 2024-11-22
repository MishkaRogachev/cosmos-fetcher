package persistence

import (
	"encoding/json"
	"os"

	"github.com/MishkaRogachev/cosmos-fetcher/protocol"
)

type BlockStore struct {
	file    *os.File
	initial bool
}

func NewBlockStore(filePath string) *BlockStore {
	// Clear the file before starting to fetch blocks
	file, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}

	// Write the opening bracket for a JSON array
	_, err = file.WriteString("[\n\t")
	if err != nil {
		panic(err)
	}

	return &BlockStore{file: file, initial: true}
}

func (bs *BlockStore) SaveBlock(block *protocol.Block) error {
	// Add a comma for all blocks except the first one
	if bs.initial {
		bs.initial = false
	} else {
		if _, err := bs.file.WriteString(",\n\t"); err != nil {
			return err
		}
	}

	data, err := json.Marshal(block)
	if err != nil {
		return err
	}

	if _, err := bs.file.Write(data); err != nil {
		return err
	}

	return nil
}

func (bs *BlockStore) Close() {
	// Write the closing bracket for a JSON array
	if _, err := bs.file.WriteString("\n]\n"); err != nil {
		panic(err)
	}

	if bs.file != nil {
		bs.file.Close()
	}
}
