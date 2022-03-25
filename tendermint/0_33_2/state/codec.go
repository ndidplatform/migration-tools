package state

import (
	cryptoamino "github.com/ndidplatform/migration-tools/tendermint/0_33_2/crypto/encoding/amino"
	amino "github.com/tendermint/go-amino"
)

var cdc = amino.NewCodec()

func init() {
	cryptoamino.RegisterAmino(cdc)
}
