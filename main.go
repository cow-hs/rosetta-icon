package main

import (
	"log"
	"net/http"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/rosetta-api/icx"
	"github.com/rosetta-api/services"
)

func NewRosettaRouter(
		network *types.NetworkIdentifier,
		asserter *asserter.Asserter,
	) http.Handler {
		client := icx.NewRpcClient()
		blockAPIService := services.NewBlockAPIService(client, network)
		blockAPIController := server.NewBlockAPIController(
			blockAPIService,
			asserter,
		)
		return server.NewRouter(blockAPIController)
}


func main() {
	networkConfig := &types.NetworkIdentifier{
		Blockchain: "ICON",
		Network: "testnet",
	}

	asserter, err := asserter.NewServer(
		[]string{"Transfer", "Reward"},
		false,
		[]*types.NetworkIdentifier{networkConfig},
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}

	router := NewRosettaRouter(networkConfig, asserter)
	loggedRouter := server.LoggerMiddleware(router)
	corsRouter := server.CorsMiddleware(loggedRouter)
	log.Fatal(http.ListenAndServe(":8080", corsRouter))
}
