// Copyright 2020 ICON Foundation, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package icon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/coinbase/rosetta-sdk-go/types"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

// Client is used to fetch blocks from ICON Node and
// to parse ICON block data into Rosetta types.
//
// We opted not to use existing ICON RPC libraries
// because they don't allow providing context
// in each request.

type Client struct {
	baseURL    string
	debugURL   string
	currency   *types.Currency
	httpClient *http.Client
}

func NewClient(
	baseURL string,
	debugURL string,
	currency *types.Currency,
) *Client {
	return &Client{
		baseURL:    baseURL,
		debugURL:   debugURL,
		currency:   currency,
		httpClient: newHTTPClient(defaultTimeout),
	}
}

// newHTTPClient returns a new HTTP client
func newHTTPClient(
	timeout time.Duration,
) *http.Client {
	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: dialTimeout,
		}).Dial,
	}
	httpClient := &http.Client{
		Timeout:   timeout,
		Transport: netTransport,
	}
	return httpClient
}

func (ic *Client) GetBlock(
	ctx context.Context,
	blockIdentifier *types.PartialBlockIdentifier,
) (*Block, error) {
	response := &blockResponse{}
	err := ic.post(ctx, requestMethodGetBlock, blockIdentifier, response)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get block", err)
	}
	return response.Result, nil
}

// post makes a HTTP request to a Bitcoin node
func (ic *Client) post(
	ctx context.Context,
	method requestMethod,
	params interface{},
	response jSONRPCResponse,
) error {
	rpcRequest := &RPCRequest{
		JSONRPC: jSONRPCVersion,
		ID:      requestID,
		Method:  string(method),
	}

	if params != nil {
		rawParams, onError := json.Marshal(params)
		if onError != nil {
			return fmt.Errorf("%w: error marshalling RPC params", onError)
		}
		rpcRequest.Params = rawParams
	}

	requestBody, err := json.Marshal(rpcRequest)
	if err != nil {
		return fmt.Errorf("%w: error marshalling RPC request", err)
	}

	req, err := http.NewRequest(http.MethodPost, ic.baseURL, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("%w: error constructing request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Perform the post request
	res, err := ic.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("%w: error posting to rpc-api", err)
	}
	defer res.Body.Close()

	// We expect JSON-RPC responses to return `200 OK` statuses
	if res.StatusCode != http.StatusOK {
		val, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("invalid response: %s %s", res.Status, string(val))
	}

	rpcResponse := &RPCResponse{}
	if err = json.NewDecoder(res.Body).Decode(rpcResponse); err != nil {
		return fmt.Errorf("%w: error decoding response body", err)
	}

	// Handle errors that are returned in JSON-RPC responses with `200 OK` statuses
	return response.Err()
}
