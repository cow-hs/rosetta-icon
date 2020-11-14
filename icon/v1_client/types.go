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
	"encoding/json"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/icon-project/goloop/server/jsonrpc"
)

type Block struct {
	BlockHash              jsonrpc.HexBytes  `json:"block_hash" validate:"required,t_hash"`
	Version                jsonrpc.HexInt    `json:"version" validate:"required,t_int"`
	Height                 int64             `json:"height" validate:"required,t_int"`
	Timestamp              int64             `json:"time_stamp" validate:"required,t_int"`
	Proposer               jsonrpc.HexBytes  `json:"peer_id" validate:"optional,t_addr_eoa"`
	PrevID                 jsonrpc.HexBytes  `json:"prev_block_hash" validate:"required,t_hash"`
	NormalTransactionsHash jsonrpc.HexBytes  `json:"merkle_tree_root_hash" validate:"required,t_hash"`
	Signature              jsonrpc.HexBytes  `json:"signature" validate:"optional,t_hash"`
	NormalTransactions     []json.RawMessage `json:"confirmed_transaction_list" `
}

type NormalTransaction struct {
	TxHash    jsonrpc.HexBytes `json:"txHash"`
	Version   jsonrpc.HexInt   `json:"version"`
	From      jsonrpc.Address  `json:"from"`
	To        jsonrpc.Address  `json:"to"`
	Value     jsonrpc.HexInt   `json:"value,omitempty" `
	StepLimit jsonrpc.HexInt   `json:"stepLimit"`
	TimeStamp jsonrpc.HexInt   `json:"timestamp"`
	NID       jsonrpc.HexInt   `json:"nid,omitempty"`
	Nonce     jsonrpc.HexInt   `json:"nonce,omitempty"`
	Signature jsonrpc.HexBytes `json:"signature"`
	DataType  string           `json:"dataType,omitempty"`
	Data      json.RawMessage  `json:"data,omitempty"`
}

type TransactionResult struct {
	To                 jsonrpc.Address  `json:"to"`
	CumulativeStepUsed jsonrpc.HexInt   `json:"cumulativeStepUsed"`
	StepUsed           jsonrpc.HexInt   `json:"stepUsed"`
	StepPrice          jsonrpc.HexInt   `json:"stepPrice"`
	EventLogs          []EventLog       `json:"eventLogs"`
	LogsBloom          jsonrpc.HexBytes `json:"logsBloom"`
	Status             jsonrpc.HexInt   `json:"status"`
	Failure            *FailureReason   `json:"failure,omitempty"`
	SCOREAddress       jsonrpc.Address  `json:"scoreAddress,omitempty"`
	BlockHash          jsonrpc.HexBytes `json:"blockHash" validate:"required,t_hash"`
	BlockHeight        jsonrpc.HexInt   `json:"blockHeight" validate:"required,t_int"`
	TxIndex            jsonrpc.HexInt   `json:"txIndex" validate:"required,t_int"`
	TxHash             jsonrpc.HexBytes `json:"txHash" validate:"required,t_int"`
	StepDetails        interface{}      `json:"stepUsedDetails,omitempty"`
}

type EventLog struct {
	Addr    jsonrpc.Address `json:"scoreAddress"`
	Indexed []*string       `json:"indexed"`
	Data    []*string       `json:"data"`
}

type FailureReason struct {
	CodeValue    jsonrpc.HexInt `json:"code"`
	MessageValue string         `json:"message"`
}

type Transaction struct {
	NormalTransaction
	BlockHash   jsonrpc.HexBytes `json:"blockHash" validate:"required,t_hash"`
	BlockHeight jsonrpc.HexInt   `json:"blockHeight" validate:"required,t_int"`
	TxIndex     jsonrpc.HexInt   `json:"txIndex" validate:"required,t_int"`
}

// ========== Extensions ========== //
type BlockMetadata struct {
	Version                jsonrpc.HexInt   `json:"version" validate:"required,t_int"`
	NormalTransactionsHash jsonrpc.HexBytes `json:"merkle_tree_root_hash" validate:"required,t_hash"`
	Proposer               jsonrpc.HexBytes `json:"peer_id" validate:"optional,t_addr_eoa"`
	Signature              jsonrpc.HexBytes `json:"signature" validate:"optional,t_hash"`
}

func (block *Block) Metadata() (map[string]interface{}, error) {
	m := &BlockMetadata{
		Version:                block.Version,
		NormalTransactionsHash: block.NormalTransactionsHash,
		Proposer:               block.Proposer,
		Signature:              block.Signature,
	}

	return types.MarshalMap(m)
}
