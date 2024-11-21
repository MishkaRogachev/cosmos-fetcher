package protocol

import "encoding/json"

type SyncInfo struct {
	EarliestBlockHeight int64 `json:"earliest_block_height,string"`
	LatestBlockHeight   int64 `json:"latest_block_height,string"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

type BlockResult struct {
	Block BlockMeta `json:"block"`
}

type BlockMeta struct {
	Header BlockHeader `json:"header"`
	Data   BlockData   `json:"data"`
}

type BlockHeader struct {
	ChainID string `json:"chain_id"`
}

type BlockData struct {
	Txs []interface{} `json:"txs"`
}

func ParseSyncInfo(data map[string]interface{}, syncInfo *SyncInfo) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, syncInfo)
}
