package types

import (
	tmbytes "github.com/ndidplatform/migration-tools/tendermint/0_33_2/libs/bytes"
)

//-------------------------------------

type PartSetHeader struct {
	Total int              `json:"total"`
	Hash  tmbytes.HexBytes `json:"hash"`
}
