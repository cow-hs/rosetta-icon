package main

import (
	"encoding/json"
	"fmt"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/leeheonseung/rosetta-icon/configuration"
	"github.com/leeheonseung/rosetta-icon/icon"
	"github.com/leeheonseung/rosetta-icon/icon/v1_client"

	"io"
	"os"
)

func main() {
	cfg, err := configuration.LoadConfiguration()
	if err != nil {
		_ = fmt.Errorf("Fail")
		return
	}

	rpcClient := icon.NewClient(cfg.URL, v1_client.Currency)

	// 이렇게 할당해야하는가?
	index := int64(1)
	params := &types.PartialBlockIdentifier{
		Index: &index,
	}

	block, err := rpcClient.GetBlock(params)
	JsonPrettyPrintln(os.Stdout, block)
	fmt.Print(err)
}

func JsonPrettyPrintln(w io.Writer, v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	_, err = fmt.Fprintln(w, string(b))
	return err
}
