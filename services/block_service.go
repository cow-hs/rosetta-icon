package services

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/rosetta-api/icx"
)

type BlockAPIService struct {
	gateway *icx.RPCClient
	network *types.NetworkIdentifier
}

func NewBlockAPIService(
	gateway *icx.RPCClient,
	network *types.NetworkIdentifier,
) server.BlockAPIServicer {
	return &BlockAPIService{
		gateway: gateway,
		network: network,
	}
}

func (s *BlockAPIService) Block(
	context context.Context,
	request *types.BlockRequest,
) (*types.BlockResponse, *types.Error) {
	blk, _ := s.gateway.GetLastBlock()
	return blk.ToRosettaResponse()
}

func (s *BlockAPIService) BlockTransaction(
	context context.Context,
	request *types.BlockTransactionRequest,
) (*types.BlockTransactionResponse, *types.Error) {
	return nil, nil
}

