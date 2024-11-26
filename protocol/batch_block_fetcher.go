package protocol

import (
	"fmt"
	"log"
	"sort"
	"sync"
)

const BLOCK_PER_BATCH = 16

type BlockBatch struct {
	StartBlockHeight int64
	EndBlockHeight   int64
	Blocks           []*Block
}

type BatchBlockFetcher struct {
	BlockFetcher
	startBlockHeight int64
	endBlockHeight   int64
	numWorkers       int
	BatchChannel     chan *BlockBatch
	wg               sync.WaitGroup
	done             chan struct{}
	quit             chan struct{}
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
		done:             make(chan struct{}),
	}
}

func (bbf *BatchBlockFetcher) sendBatch(startBlockHeight, endBlockHeight int64, blocks []*Block) {
	// Sort blocks by height before sending
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].BlockHeight < blocks[j].BlockHeight
	})
	bbf.BatchChannel <- &BlockBatch{
		Blocks:           blocks,
		StartBlockHeight: startBlockHeight,
		EndBlockHeight:   endBlockHeight,
	}
}

func (bbf *BatchBlockFetcher) fetchBlocksForWorkerID(workerID int) {
	// Notify that this worker is done when the function returns
	bbf.wg.Add(1)
	defer bbf.wg.Done()

	blocks := []*Block{}
	for batchStart := bbf.startBlockHeight + int64(workerID)*BLOCK_PER_BATCH; batchStart <= bbf.endBlockHeight; batchStart += int64(bbf.numWorkers * BLOCK_PER_BATCH) {
		batchEnd := batchStart + BLOCK_PER_BATCH - 1
		if batchEnd > bbf.endBlockHeight {
			batchEnd = bbf.endBlockHeight
		}

		fmt.Println("Worker", workerID, "starts to fetch blocks", batchStart, "to", batchEnd)

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
				blocks = append(blocks, block)
			}
		}

		// Send the completed batch if it's not empty
		if len(blocks) > 0 {
			bbf.sendBatch(batchStart, batchEnd, blocks)
			blocks = []*Block{}
		}
	}
	fmt.Println("Worker", workerID, "finished")
}

func (bbf *BatchBlockFetcher) StartFetchingBlocks() {
	for i := 0; i < bbf.numWorkers; i++ {
		go bbf.fetchBlocksForWorkerID(i)
	}

	// Wait for all workers to complete, then close the BatchChannel and signal done
	go func() {
		bbf.wg.Wait()
		close(bbf.BatchChannel)
		close(bbf.done)
	}()
}

func (bbf *BatchBlockFetcher) StopFetchingBlocks() {
	close(bbf.quit)
	bbf.wg.Wait()
}

func (bbf *BatchBlockFetcher) WaitDone() <-chan struct{} {
	return bbf.done
}
