package store

import (
	"github.com/ndidplatform/migration-tools/tendermint/0_33_2/types"
	amino "github.com/tendermint/go-amino"
)

var cdc = amino.NewCodec()

func init() {
	types.RegisterBlockAmino(cdc)
}
