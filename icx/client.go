package icx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rosetta-api/types"
)

const (
	icxEndpointURL = "https://zicon.net.solidwallet.io/api/v3"
)

type RPCRequest struct {
	Method 	string			`json:"method"`
	ID 		int				`json:"id"`
	JSONRPC string			`json:"jsonrpc"`
	Params 	json.RawMessage `json:"params,omitempty"`
}

type RPCResponse struct {
	JSONRPC string			`json:"jsonrpc"`
	Result 	json.RawMessage	`json:"result"`
	Error	int				`json:"error,omitempty"`
	ID		int				`json:"id"`
}

type RPCClient struct {
	endpoint string
	hc *http.Client
}

func NewRpcClient() *RPCClient {
	return &RPCClient{
		endpoint: icxEndpointURL,
		hc: &http.Client{},
	}
}

func (r *RPCClient) DoCall(method string, params interface{}, result interface{}) (response *RPCResponse, err error) {
	request := &RPCRequest{
		Method: method,
		ID: 1234,
		JSONRPC: "2.0",
	}

	if params != nil {
		rawParams, onError := json.Marshal(params)
		if onError != nil {
			err = onError
			return
		}
		request.Params = json.RawMessage(rawParams)
	}

	body, _ := json.Marshal(request)
	req, err := http.NewRequest("POST", "https://bicon.net.solidwallet.io/api/v3", bytes.NewReader(body))
	if err != nil {
		fmt.Println(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, _ := r.Call(req)
	rpcResp, _ := decodeBody(resp)

	if result != nil {
		err = json.Unmarshal(rpcResp.Result, result)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	return
}

func (r *RPCClient) Call(req *http.Request) (response *http.Response, err error) {
	response, err = r.hc.Do(req)
	if response.StatusCode != 200 {
		fmt.Println("RPC Response: ", response.StatusCode)
		return
	}
	return
}

func (r *RPCClient) GetLastBlock() (block *types.Block, err error) {
	block = &types.Block{}
	_, err = r.DoCall("icx_getLastBlock", nil, block)
	if err != nil {
		return nil, err
	}
	return
}

func decodeBody(resp *http.Response) (rpcResponse *RPCResponse, err error) {
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&rpcResponse)
	return
}

