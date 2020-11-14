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

package v1_client

import (
	"github.com/icon-project/goloop/server/jsonrpc"
	v3 "github.com/icon-project/goloop/server/v3"
	"net/http"
	"net/url"
	"strings"
)

type ClientV3 struct {
	*JsonRpcClient
	DebugEndPoint string
}

func guessDebugEndpoint(endpoint string) string {
	uo, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}
	ps := strings.Split(uo.Path, "/")
	for i, v := range ps {
		if v == "api" {
			if len(ps) > i+1 && ps[i+1] == "v3" {
				ps[i+1] = "v3d"
				uo.Path = strings.Join(ps, "/")
				return uo.String()
			}
			break
		}
	}
	return ""
}

func NewClientV3(endpoint string) *ClientV3 {
	client := new(http.Client)
	apiClient := NewJsonRpcClient(client, endpoint)

	return &ClientV3{
		JsonRpcClient: apiClient,
		DebugEndPoint: guessDebugEndpoint(endpoint),
	}
}

func (c *ClientV3) GetLastBlock() (*Block, error) {
	blk := &Block{}
	_, err := c.Do("icx_getLastBlock", nil, blk)
	if err != nil {
		return nil, err
	}
	return blk, nil
}

func (c *ClientV3) GetBlockByHeight(param *v3.BlockHeightParam) (*Block, error) {
	blk := &Block{}
	_, err := c.Do("icx_getBlockByHeight", param, blk)
	if err != nil {
		return nil, err
	}
	return blk, nil
}

func (c *ClientV3) GetBlockByHash(param *v3.BlockHashParam) (*Block, error) {
	blk := &Block{}
	_, err := c.Do("icx_getBlockByHash", param, blk)
	if err != nil {
		return nil, err
	}
	return blk, nil
}

func (c *ClientV3) Call(param *v3.CallParam) (interface{}, error) {
	var result interface{}
	_, err := c.Do("icx_call", param, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ClientV3) GetBalance(param *v3.AddressParam) (*jsonrpc.HexInt, error) {
	var result jsonrpc.HexInt
	_, err := c.Do("icx_getBalance", param, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *ClientV3) GetScoreApi(param *v3.ScoreAddressParam) ([]interface{}, error) {
	var result []interface{}
	_, err := c.Do("icx_getScoreApi", param, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ClientV3) GetTotalSupply() (*jsonrpc.HexInt, error) {
	var result jsonrpc.HexInt
	_, err := c.Do("icx_getTotalSupply", nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *ClientV3) GetTransactionResult(param *v3.TransactionHashParam) (*TransactionResult, error) {
	tr := &TransactionResult{}
	_, err := c.Do("icx_getTransactionResult", param, tr)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

func (c *ClientV3) WaitTransactionResult(param *v3.TransactionHashParam) (*TransactionResult, error) {
	tr := &TransactionResult{}
	if _, err := c.Do("icx_waitTransactionResult", param, tr); err != nil {
		return nil, err
	}
	return tr, nil
}

func (c *ClientV3) GetTransactionByHash(param *v3.TransactionHashParam) (*Transaction, error) {
	t := &Transaction{}
	_, err := c.Do("icx_getTransactionByHash", param, t)
	if err != nil {
		return nil, err
	}
	return t, nil
}
