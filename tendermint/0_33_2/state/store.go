package state

import (
	"fmt"

	tmos "github.com/ndidplatform/migration-tools/tendermint/0_33_2/libs/os"
	dbm "github.com/tendermint/tm-db"
)

// LoadState loads the State from the database.
func LoadState(db dbm.DB) State {
	return loadState(db, stateKey)
}

func loadState(db dbm.DB, key []byte) (state State) {
	buf, err := db.Get(key)
	if err != nil {
		panic(err)
	}
	if len(buf) == 0 {
		return state
	}

	err = cdc.UnmarshalBinaryBare(buf, &state)
	if err != nil {
		// DATA HAS BEEN CORRUPTED OR THE SPEC HAS CHANGED
		tmos.Exit(fmt.Sprintf(`LoadState: Data has been corrupted or its spec has changed:
                %v\n`, err))
	}
	// TODO: ensure that buf is completely read.

	return state
}
