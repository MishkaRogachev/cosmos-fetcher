package protocol

import (
	"log"
	"sort"
	"sync"
)

const BLOCK_PER_BATCH = 16

type BlockBatch struct {
	Blocks []*Block
}

type BatchBlockFetcher struct {
	BlockFetcher
	startBlockHeight int64
	endBlockHeight   int64
	numWorkers       int
	BatchChannel     chan *BlockBatch
	quit             chan struct{}
	wg               sync.WaitGroup
}

func NewBatchBlockFetcher(client *RPCClient, startBlockHeight, endBlockHeight int64, numWorkers int) *BatchBlockFetcher {
	blockFetcher := NewBlockFetcher(client)
	return &BatchBlockFetcher{
		BlockFetcher:     *blockFetcher,
		startBlockHeight: startBlockHeight,
		endBlockHeight:   endBlockHeight,
		numWorkers:       numWorkers,
		BatchChannel:     make(chan *BlockBatch),
		quit:             make(chan struct{}),
	}
}

func (bbf *BatchBlockFetcher) sendBatch(batch []*Block) {
	// Sort blocks by height before sending
	sort.Slice(batch, func(i, j int) bool {
		return batch[i].BlockHeight < batch[j].BlockHeight
	})
	bbf.BatchChannel <- &BlockBatch{Blocks: batch}
}

func (bbf *BatchBlockFetcher) fetchBlocksForWorkerID(workerID int) {
	batch := []*Block{}
	for batchStart := bbf.startBlockHeight + int64(workerID)*BLOCK_PER_BATCH; batchStart <= bbf.endBlockHeight; batchStart += int64(bbf.numWorkers * BLOCK_PER_BATCH) {
		batchEnd := batchStart + BLOCK_PER_BATCH - 1
		if batchEnd > bbf.endBlockHeight {
			batchEnd = bbf.endBlockHeight
		}

		for height := batchStart; height <= batchEnd; height++ {
			select {
			case <-bbf.quit:
				return
			default:
				block, err := bbf.FetchBlock(height)
				if err != nil {
					log.Printf("Error fetching block at height %d: %v", height, err)
					// TODO: retry fetching the block
					continue
				}
				batch = append(batch, block)
			}
		}

		// Send the completed batch if it's not empty
		if len(batch) > 0 {
			bbf.sendBatch(batch)
			batch = []*Block{}
		}
	}
}

func (bbf *BatchBlockFetcher) StartFetchingBlocks() {
	for i := 0; i < bbf.numWorkers; i++ {
		bbf.wg.Add(1)
		go func(workerID int) {
			defer bbf.wg.Done()
			bbf.fetchBlocksForWorkerID(workerID)
		}(i)
	}
}

func (bbf *BatchBlockFetcher) StopFetchingBlocks() {
	close(bbf.quit)
	bbf.wg.Wait()
	close(bbf.BatchChannel)
}
