package protocol

import (
	"encoding/json"
	"fmt"
	"io"

	"net/http"
)

type RPCClient struct {
	RPCURL     string
	httpClient *http.Client
}

func NewRPCClient(rpcURL string, httpClient *http.Client) *RPCClient {
	return &RPCClient{RPCURL: rpcURL, httpClient: httpClient}
}

func (client *RPCClient) getBody(url string) ([]byte, error) {
	resp, err := client.httpClient.Get(url)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}
	return body, nil
}

func (client *RPCClient) SyncInfo() (SyncInfo, error) {
	body, err := client.getBody(fmt.Sprintf("%s/status", client.RPCURL))
	if err != nil {
		return SyncInfo{}, err
	}

	var status RPCStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return SyncInfo{}, err
	}

	if status.Error != nil {
		return SyncInfo{}, fmt.Errorf("RPC error: %s", status.Error.Message)
	}

	return status.Result.SyncInfo, nil
}

func (client *RPCClient) BlockHeight(height int64) (BlockResult, error) {
	body, err := client.getBody(fmt.Sprintf("%s/block?height=%d", client.RPCURL, height))
	if err != nil {
		return BlockResult{}, err
	}

	var response BlockResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return BlockResult{}, err
	}

	if response.Error != nil {
		return BlockResult{}, fmt.Errorf("RPC error: %s", response.Error.Message)
	}
	return *response.Result, nil
}
