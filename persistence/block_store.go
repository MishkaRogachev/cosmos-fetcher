package persistence

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/MishkaRogachev/cosmos-fetcher/protocol"
)

type BlockStore struct {
	filePath string
	mutex    sync.Mutex
}

func NewBlockStore(filePath string) *BlockStore {
	return &BlockStore{filePath: filePath}
}

func (bs *BlockStore) SaveBlock(block *protocol.Block) error {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	file, err := os.OpenFile(bs.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(block)
	if err != nil {
		return err
	}

	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}

	return nil
}
