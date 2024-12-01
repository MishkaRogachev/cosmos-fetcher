package protocol

import "encoding/json"

type SyncInfo struct {
	EarliestBlockHeight int64 `json:"earliest_block_height,string"`
	LatestBlockHeight   int64 `json:"latest_block_height,string"`
}

type RPCStatus struct {
	Result *struct {
		SyncInfo SyncInfo `json:"sync_info"`
	} `json:"result"`
	Error *RPCError `json:"error"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

type BlockResponse struct {
	Result *BlockResult `json:"result"`
	Error  *RPCError    `json:"error"`
}

type BlockResult struct {
	BlockID BlockID   `json:"block_id"`
	Block   BlockMeta `json:"block"`
}

type BlockID struct {
	Hash  string    `json:"hash"`
	Parts PartsMeta `json:"parts"`
}

type BlockMeta struct {
	Header BlockHeader `json:"header"`
	Data   BlockData   `json:"data"`
}

type PartsMeta struct {
	Total int    `json:"total"`
	Hash  string `json:"hash"`
}

type BlockHeader struct {
	ChainID string `json:"chain_id"`
	Height  string `json:"height"`
	Time    string `json:"time"`
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
