// Copied and modified from tendermint/blockchain/store.go

package block

import (
	"fmt"

	"github.com/pkg/errors"
	dbm "github.com/tendermint/tm-db"
)

func calcBlockMetaKey(height int64) []byte {
	return []byte(fmt.Sprintf("H:%v", height))
}

// LoadBlockMeta returns the BlockMeta for the given height.
// If no block is found for the given height, it returns nil.
func LoadBlockMeta(db dbm.DB, height int64) (*BlockMeta, error) {
	var blockMeta = new(BlockMeta)
	bz, err := db.Get(calcBlockMetaKey(height))
	if err != nil {
		return nil, err
	}
	if len(bz) == 0 {
		return nil, nil
	}
	err = cdc.UnmarshalBinaryBare(bz, blockMeta)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading block meta")
	}
	return blockMeta, nil
}
