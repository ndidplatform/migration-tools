package store

import (
	"fmt"

	"github.com/pkg/errors"

	dbm "github.com/tendermint/tm-db"

	"github.com/ndidplatform/migration-tools/tendermint/0_33_2/types"
)

func calcBlockMetaKey(height int64) []byte {
	return []byte(fmt.Sprintf("H:%v", height))
}

// LoadBlockMeta returns the BlockMeta for the given height.
// If no block is found for the given height, it returns nil.
func LoadBlockMeta(db dbm.DB, height int64) *types.BlockMeta {
	var blockMeta = new(types.BlockMeta)
	bz, err := db.Get(calcBlockMetaKey(height))
	if err != nil {
		panic(err)
	}
	if len(bz) == 0 {
		return nil
	}
	err = cdc.UnmarshalBinaryBare(bz, blockMeta)
	if err != nil {
		panic(errors.Wrap(err, "Error reading block meta"))
	}
	return blockMeta
}
