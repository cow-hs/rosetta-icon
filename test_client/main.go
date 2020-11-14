package main

import (
	"encoding/json"
	"fmt"
	"github.com/leeheonseung/rosetta-icon/configuration"
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

	rpcClient := v1_client.NewClientV3(cfg.URL)
	blk, err := rpcClient.GetLastBlock()
	JsonPrettyPrintln(os.Stdout, blk)
}

func JsonPrettyPrintln(w io.Writer, v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	_, err = fmt.Fprintln(w, string(b))
	return err
}
