package main

import (
	"context"
	"fmt"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/leeheonseung/rosetta-icon/configuration"
	"github.com/leeheonseung/rosetta-icon/icon"
	"log"
)

func main() {
	ctx := context.Background()

	cfg, err := configuration.LoadConfiguration()
	if err != nil {
		_ = fmt.Errorf("Fail")
		return
	}

	partialBlock := &types.PartialBlockIdentifier{}
	client := icon.NewClient(cfg.URL, cfg.DebugURL, icon.Currency)
	block, err := client.GetBlock(ctx, partialBlock)
	log.Print(block, err)
}
