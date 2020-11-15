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
	"errors"
	"fmt"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/icon-project/goloop/server/jsonrpc"
	"strconv"
)

func ParseBlock(raw map[string]interface{}) (*types.Block, error) {
	version := raw["version"]
	switch version {
	case "0.1a":
		return Block_0_1a(raw)
	case "0.3", "0.4", "0.5":
		return Block_0_3(raw)
	}
	return nil, errors.New("Unsupported Block Version")
}

func Block_0_1a(raw map[string]interface{}) (*types.Block, error) {
	meta := map[string]interface{}{
	"version": 					raw["version"],
	"peer_id":					raw["peer_id"],
	"signature":				raw["signature"],
	"next_leader":				raw["next_leader"],
	"merkle_tree_root_hash": 	raw["merkle_tree_root_hash"],
	}

	// 꼭 이렇게 한번 더 할당해야하는가?
	txs, _ := ParseTransaction(raw["confirmed_transaction_list"].([]interface{}))
	fmt.Print(txs)

	index, _ := strconv.ParseInt(raw["height"].(string), 10, 64)
	timestamp, _ := strconv.ParseInt(raw["time_stamp"].(string), 10, 64) // 1000

	if index == genesisBlockIndex {
		return &types.Block{
			BlockIdentifier: &types.BlockIdentifier{
				Index: 		index,
				Hash:  		raw["block_hash"].(string),
			},
			Timestamp: 		timestamp,
			Transactions: 	nil,
			Metadata:		meta,
		}, nil
	} else {
		return &types.Block{
			BlockIdentifier: &types.BlockIdentifier{
				Index: 		index,
				Hash:  		raw["block_hash"].(string),
			},
			ParentBlockIdentifier: &types.BlockIdentifier{
				Index: 		index - 1,
				Hash:  		raw["prev_block_hash"].(string),
			},
			Timestamp: 		timestamp,
			Transactions: 	nil,
			Metadata:		meta,
		}, nil
	}
}

func Block_0_3(raw map[string]interface{}) (*types.Block, error) {
	meta := map[string]interface{}{
		"version": 				raw["version"],
		"transactionsHash":		raw["transactionsHash"],
		"stateHash":			raw["stateHash"],
		"receiptsHash":			raw["receiptsHash"],
		"repsHash": 			raw["repsHash"],
		"nextRepsHash": 		raw["nextRepsHash"],
		"leaderVotesHash": 		raw["leaderVotesHash"],
		"prevVotesHash": 		raw["prevVotesHash"],
		"logsBloom": 			raw["logsBloom"],
		"leaderVotes": 			raw["leaderVotes"],
		"prevVotes": 			raw["prevVotes"],
		"leader": 				raw["leader"],
		"signature": 			raw["signature"],
		"nextLeader": 			raw["nextLeader"],
	}

	txs, _ := ParseTransaction(raw["confirmed_transaction_list"].([]interface{}))
	fmt.Print(txs)

	index := jsonrpc.HexInt(raw["height"].(string)).Value()
	timestamp := jsonrpc.HexInt(raw["timestamp"].(string)).Value() // 1000

	if index == genesisBlockIndex {
		return &types.Block{
			BlockIdentifier: &types.BlockIdentifier{
				Index: 		index,
				Hash:  		raw["hash"].(string),
			},
			Timestamp: 		timestamp,
			Transactions: 	nil,
			Metadata:		meta,
		}, nil
	} else {
		return &types.Block{
			BlockIdentifier: &types.BlockIdentifier{
				Index: 		index,
				Hash:  		raw["hash"].(string),
			},
			ParentBlockIdentifier: &types.BlockIdentifier{
				Index: 		index - 1,
				Hash:  		raw["prevHash"].(string),
			},
			Timestamp: 		timestamp,
			Transactions: 	nil,
			Metadata:		meta,
		}, nil
	}
}

func ParseTransaction(raw []interface{}) (*[]types.Transaction, error) {
	var transactions []types.Transaction
	
	for index, transaction := range raw {
		version := transaction.(map[string]interface{})["version"]
		switch version {
		case 3:
			return nil, nil
		default :
			tx, _ := TransactionV2(int64(index), transaction.(map[string]interface{}))
			transactions = append(transactions, *tx)
		}
		println(index, transaction)
	}
	//switch version {
	//case 3:
	//	return nil, nil
	//default :
	//	return TransactionV2(raw)
	//}
	//return nil, errors.New("Unsupported Transaction Version")
	return nil, nil
}

func TransactionV2(index int64, raw map[string]interface{}) (*types.Transaction, error) {
	meta := map[string]interface{}{
	}

	operation := types.Operation{
		OperationIdentifier: &types.OperationIdentifier{
			Index: index,
			NetworkIndex: nil,
		},
		RelatedOperations: nil,
		Type: raw["transfer"].(string),
		Status: nil,
		Account: &types.AccountIdentifier{
			Address:
		}
	}

	return &types.Transaction{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: raw["tx_hash"].(string),
		},
		Operations: operation,
		Metadata: meta,
	}, nil
}