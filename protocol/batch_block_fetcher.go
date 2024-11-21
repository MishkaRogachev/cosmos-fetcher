package protocol

import (
	"sync"
)

type BatchBlockFetcher struct {
	BlockFetcher
	startBlockHeight int64
	endBlockHeight   int64
	numWorkers       int
	blockChannel     chan *Block
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
		blockChannel:     make(chan *Block),
		quit:             make(chan struct{}),
	}
}

func (bbf *BatchBlockFetcher) StartFetchingBlocks() chan *Block {
	for i := 0; i < bbf.numWorkers; i++ {
		bbf.wg.Add(1)
		go func(workerID int) {
			defer bbf.wg.Done()
			for height := bbf.startBlockHeight + int64(workerID); height <= bbf.endBlockHeight; height += int64(bbf.numWorkers) {
				select {
				case <-bbf.quit:
					return
				default:
					block, err := bbf.FetchBlock(height)
					if err != nil {
						continue
					}
					bbf.blockChannel <- block
				}
			}
		}(i)
	}

	go func() {
		bbf.wg.Wait()
		close(bbf.blockChannel)
	}()

	return bbf.blockChannel
}

func (bbf *BatchBlockFetcher) StopFetchingBlocks() {
	close(bbf.quit)
	bbf.wg.Wait()
}
