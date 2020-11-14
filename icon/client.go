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
	"fmt"
	RosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	icon_v1 "github.com/leeheonseung/rosetta-icon/icon/v1_client"
)

// Client is used to fetch blocks from ICON Node and
// to parse ICON block data into Rosetta types.
//
// We opted not to use existing ICON RPC libraries
// because they don't allow providing context
// in each request.

type Client struct {
	currency *RosettaTypes.Currency
	iconV1   *icon_v1.ClientV3
}

func NewClient(
	endpoint string,
	currency *RosettaTypes.Currency,
) *Client {
	return &Client{
		currency,
		icon_v1.NewClientV3(endpoint),
	}
}

func (ic *Client) GetLastBlock() (*RosettaTypes.Block, error) {

	block, err := ic.iconV1.GetLastBlock()
	if err != nil {
		return nil, fmt.Errorf("%w: could not get last block", err)
	}

	blockIdentifier := &RosettaTypes.BlockIdentifier{
		Hash:  string(block.BlockHash),
		Index: block.Height,
	}

	parentBlockIdentifier := blockIdentifier

	if blockIdentifier.Index != genesisBlockIndex {
		parentBlockIdentifier = &RosettaTypes.BlockIdentifier{
			Hash:  string(block.PrevID),
			Index: blockIdentifier.Index - 1,
		}
	}

	metadata, err := block.Metadata()
	if err != nil {
		return nil, fmt.Errorf("%w: could not get block metadata", err)
	}

	return &RosettaTypes.Block{
		BlockIdentifier:       blockIdentifier,
		ParentBlockIdentifier: parentBlockIdentifier,
		Timestamp:             convertTime(block.Timestamp),
		Transactions:          txs,
		Metadata:              metadata,
	}, nil
}
