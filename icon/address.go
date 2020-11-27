package icon

import (
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/icon-project/goloop/common"
	"github.com/leeheonseung/rosetta-icon/services"
)

const AddressStringLength = 42

func CheckAddress(as string) *types.Error {
	if len(as) != AddressStringLength {
		return services.ErrInvalidAddress
	}
	a := new(common.Address)
	if a.SetString(as) != nil {
		return services.ErrInvalidAddress
	}
	return nil
}
