package protocol

import (
	"fmt"
	"log"
	"sort"
	"sync"
)

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
	blocksPerBatch   int
	BatchChannel     chan *BlockBatch
	wg               sync.WaitGroup
	done             chan struct{}
	quit             chan struct{}
}

func NewBatchBlockFetcher(client *RPCClient, startBlockHeight, endBlockHeight int64, numWorkers, blocksPerBatch, maxRetries, retryInterval int) *BatchBlockFetcher {
	blockFetcher := NewBlockFetcher(client, maxRetries, retryInterval)
	return &BatchBlockFetcher{
		BlockFetcher:     *blockFetcher,
		startBlockHeight: startBlockHeight,
		endBlockHeight:   endBlockHeight,
		numWorkers:       numWorkers,
		blocksPerBatch:   blocksPerBatch,
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
	for batchStart := bbf.startBlockHeight + int64(workerID*bbf.blocksPerBatch); batchStart <= bbf.endBlockHeight; batchStart += int64(bbf.numWorkers * bbf.blocksPerBatch) {
		batchEnd := batchStart + int64(bbf.blocksPerBatch) - 1
		if batchEnd > bbf.endBlockHeight {
			batchEnd = bbf.endBlockHeight
		}

		fmt.Println("Worker", workerID, "starts to fetch blocks", batchStart, "to", batchEnd)

		for height := batchStart; height <= batchEnd; height++ {
			select {
			case <-bbf.quit:
				return
			default:
				block, err := bbf.FetchBlockWithRetries(height)
				if err != nil {
					// Log the error but continue with the next blocks
					log.Printf("Skip block %d because of error: %v", height, err)
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
