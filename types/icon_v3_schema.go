package types

import (
	"encoding/json"

	rsType "github.com/coinbase/rosetta-sdk-go/types"
)

type HexString string
type HexInt string
type Address string

type Block struct {
	BlockHash		HexString 			`json:"block_hash" validate:"required"`
	Version			HexInt 				`json:"version" validate:"required"`
	Height			int64 				`json:"height" validate:"required"`
	Timestamp		int64				`json:"time_stamp" validate:"required"`
	Creator			HexString			`json:"peer_id" validate:"optional"`
	PreviousHash	HexString			`json:"prev_block_hash" validate:"required"`
	MerkleHash		HexString			`json:"merkle_tree_root_hash" validate:"required"`
	Signature		HexString			`json:"signature" validate:"optional"`
	Transactions	[]Transaction		`json:"confirmed_transaction_list" validate:"optional"`
}

type Transaction struct {
	TransactionHash	HexString		`json:"txHash"`
	Version   		HexInt   		`json:"version"`
	From			Address  		`json:"from"`
	To        		Address  		`json:"to"`
	Value     		HexInt   		`json:"value,omitempty" `
	StepLimit		HexInt  		`json:"stepLimit"`
	TimeStamp 		HexInt   		`json:"timestamp"`
	NID       		HexInt   		`json:"nid,omitempty"`
	Nonce     		HexInt   		`json:"nonce,omitempty"`
	Signature 		HexString 		`json:"signature"`
	DataType  		string      	`json:"dataType,omitempty"`
	Data      		json.RawMessage	`json:"data,omitempty"`
}

type TransactionResponse struct {
	Transaction
	BlockHash	HexString	`json:"blockHash"`
	BlockHeight	HexInt		`json:"blockHeight"`
	TxIndex		HexInt		`json:"txIndex"`
}

func (b *Block) ToRosettaResponse() (*rsType.BlockResponse, *rsType.Error) {
	return &rsType.BlockResponse{
		Block: &rsType.Block{
			BlockIdentifier: &rsType.BlockIdentifier{
				Index: b.Height,
				Hash:  string(b.BlockHash),
			},
			ParentBlockIdentifier: &rsType.BlockIdentifier{
				Index: b.Height - 1,
				Hash:  string(b.PreviousHash),
			},
			Timestamp:    b.Timestamp,
			Transactions: b.ToRosettaTransactionResponse(),
		},
	}, nil
}

func (b *Block) ToRosettaTransactionResponse() []*rsType.Transaction {
	txs := make([]*rsType.Transaction, len(b.Transactions))
	for i, t := range b.Transactions {
		txs[i], _ = t.ToRosettaResponse(i)
	}
	return txs
}

func (t *Transaction) ToRosettaResponse(i int) (*rsType.Transaction, *rsType.Error) {
	return &rsType.Transaction{
		TransactionIdentifier: &rsType.TransactionIdentifier{
			Hash: string(t.TransactionHash),
		},
		Operations: []*rsType.Operation{
			&rsType.Operation{
				OperationIdentifier: &rsType.OperationIdentifier{
					Index: int64(i),
				},
				Status: "Success",
				Type:   "Transfer",
				Account: &rsType.AccountIdentifier{
					Address: string(t.From),
				},
				Amount: &rsType.Amount{
					Value: string(t.Value),
					Currency: &rsType.Currency{
						Symbol:   "ICX",
						Decimals: 18,
					},
				},
			},
		},
	}, nil
}
