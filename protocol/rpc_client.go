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

func (client *RPCClient) Status() (map[string]interface{}, error) {
	resp, err := client.httpClient.Get(fmt.Sprintf("%s/status", client.RPCURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}
