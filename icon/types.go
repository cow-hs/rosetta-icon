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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coinbase/rosetta-sdk-go/types"
)

var (
	// Currency is the *types.Currency for all
	// ICON networks.
	Currency = &types.Currency{
		Symbol:   Symbol,
		Decimals: Decimals,
	}

	OperationTypes = []string{
		"TEST",
	}
)

type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   int             `json:"error,omitempty"`
	ID      int             `json:"id"`
}

type jSONRPCResponse interface {
	Err() error
}

type responseError struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

// ========== BLOCK ========== //
type blockResponse struct {
	Result *Block         `json:"result"`
	Error  *responseError `json:"error"`
}

func (b blockResponse) Err() error {
	if b.Error == nil {
		return nil
	}
	return fmt.Errorf(
		"%w: error JSON RPC response, code: %d, message: %s",
		errors.New("JSON-RPC error"),
		b.Error.Code,
		b.Error.Message,
	)
}

type HexString string
type HexInt string
type Address string

type Block struct {
	BlockHash    HexString     `json:"block_hash" validate:"required"`
	Version      HexInt        `json:"version" validate:"required"`
	Height       int64         `json:"height" validate:"required"`
	Timestamp    int64         `json:"time_stamp" validate:"required"`
	Creator      HexString     `json:"peer_id" validate:"optional"`
	PreviousHash HexString     `json:"prev_block_hash" validate:"required"`
	MerkleHash   HexString     `json:"merkle_tree_root_hash" validate:"required"`
	Signature    HexString     `json:"signature" validate:"optional"`
	Transactions []Transaction `json:"confirmed_transaction_list" validate:"optional"`
}

type Transaction struct {
	TransactionHash HexString       `json:"txHash"`
	Version         HexInt          `json:"version"`
	From            Address         `json:"from"`
	To              Address         `json:"to"`
	Value           HexInt          `json:"value,omitempty" `
	StepLimit       HexInt          `json:"stepLimit"`
	TimeStamp       HexInt          `json:"timestamp"`
	NID             HexInt          `json:"nid,omitempty"`
	Nonce           HexInt          `json:"nonce,omitempty"`
	Signature       HexString       `json:"signature"`
	DataType        string          `json:"dataType,omitempty"`
	Data            json.RawMessage `json:"data,omitempty"`
}

type TransactionResponse struct {
	Transaction
	BlockHash   HexString `json:"blockHash"`
	BlockHeight HexInt    `json:"blockHeight"`
	TxIndex     HexInt    `json:"txIndex"`
}

func (b *Block) ToRosettaResponse() (*types.BlockResponse, *types.Error) {
	return &types.BlockResponse{
		Block: &types.Block{
			BlockIdentifier: &types.BlockIdentifier{
				Index: b.Height,
				Hash:  string(b.BlockHash),
			},
			ParentBlockIdentifier: &types.BlockIdentifier{
				Index: b.Height - 1,
				Hash:  string(b.PreviousHash),
			},
			Timestamp:    b.Timestamp,
			Transactions: b.ToRosettaTransactionResponse(),
		},
	}, nil
}

func (b *Block) ToRosettaTransactionResponse() []*types.Transaction {
	txs := make([]*types.Transaction, len(b.Transactions))
	for i, t := range b.Transactions {
		txs[i], _ = t.ToRosettaResponse(i)
	}
	return txs
}

func (t *Transaction) ToRosettaResponse(i int) (*types.Transaction, *types.Error) {
	return &types.Transaction{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: string(t.TransactionHash),
		},
		Operations: []*types.Operation{
			{
				OperationIdentifier: &types.OperationIdentifier{
					Index: int64(i),
				},
				Status: "Success",
				Type:   "Transfer",
				Account: &types.AccountIdentifier{
					Address: string(t.From),
				},
				Amount: &types.Amount{
					Value:    string(t.Value),
					Currency: Currency,
				},
			},
		},
	}, nil
}
