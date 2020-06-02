// Copied and modified from tendermint/state/store.go

package state

import (
	"fmt"

	"github.com/pkg/errors"
	dbm "github.com/tendermint/tm-db"
)

var (
	stateKey = []byte("stateKey")
)

// LoadState loads the State from the database.
func LoadState(db dbm.DB) (*State, error) {
	return loadState(db, stateKey)
}

func loadState(db dbm.DB, key []byte) (state *State, err error) {
	buf, err := db.Get(key)
	if err != nil {
		return nil, err
	}
	if len(buf) == 0 {
		return state, nil
	}

	err = cdc.UnmarshalBinaryBare(buf, &state)
	if err != nil {
		// DATA HAS BEEN CORRUPTED OR THE SPEC HAS CHANGED
		return nil, errors.Wrap(err, fmt.Sprintf(`LoadState: Data has been corrupted or its spec has changed`))
	}
	// TODO: ensure that buf is completely read.

	return state, nil
}
