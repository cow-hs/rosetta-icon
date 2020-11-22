package main

import (
	"encoding/json"
	"fmt"
	"github.com/leeheonseung/rosetta-icon/configuration"
	"github.com/leeheonseung/rosetta-icon/icon"
	"github.com/leeheonseung/rosetta-icon/icon/client_v1"
	"io"
	"os"
)

func main() {
	cfg, err := configuration.LoadConfiguration()
	if err != nil {
		_ = fmt.Errorf("Fail")
		return
	}

	rpcClient := icon.NewClient(cfg.URL, client_v1.ICXCurrency)

	//// 이렇게 할당해야하는가?
	//index := int64(1)
	//params := &types.PartialBlockIdentifier{
	//	Index: &index,
	//}
	//
	//block, err := rpcClient.GetBlock(params)
	//err = JsonPrettyPrintln(os.Stdout, block)

	status, err := rpcClient.GetPeer()
	err = JsonPrettyPrintln(os.Stdout, status)
	fmt.Print(err)
}

func JsonPrettyPrintln(w io.Writer, v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	_, err = fmt.Fprintln(w, string(b))
	return err
}
