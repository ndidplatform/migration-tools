// LoadState loads the State from the database.
package state

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	dbm "github.com/tendermint/tm-db"

	tmos "github.com/ndidplatform/migration-tools/tendermint/0_34_19/libs/os"
	tmstate "github.com/ndidplatform/migration-tools/tendermint/0_34_19/proto/tendermint/state"
)

//----------------------

//go:generate mockery --case underscore --name Store

// Store defines the state store interface
//
// It is used to retrieve current state and save and load ABCI responses,
// validators and consensus parameters
type Store interface {
	// // LoadFromDBOrGenesisFile loads the most recent state.
	// // If the chain is new it will use the genesis file from the provided genesis file path as the current state.
	// LoadFromDBOrGenesisFile(string) (State, error)
	// // LoadFromDBOrGenesisDoc loads the most recent state.
	// // If the chain is new it will use the genesis doc as the current state.
	// LoadFromDBOrGenesisDoc(*types.GenesisDoc) (State, error)
	// Load loads the current state of the blockchain
	Load() (State, error)
	// // LoadValidators loads the validator set at a given height
	// LoadValidators(int64) (*types.ValidatorSet, error)
	// // LoadABCIResponses loads the abciResponse for a given height
	// LoadABCIResponses(int64) (*tmstate.ABCIResponses, error)
	// // LoadConsensusParams loads the consensus params for a given height
	// LoadConsensusParams(int64) (tmproto.ConsensusParams, error)
	// // Save overwrites the previous state with the updated one
	// Save(State) error
	// // SaveABCIResponses saves ABCIResponses for a given height
	// SaveABCIResponses(int64, *tmstate.ABCIResponses) error
	// // Bootstrap is used for bootstrapping state when not starting from a initial height.
	// Bootstrap(State) error
	// // PruneStates takes the height from which to start prning and which height stop at
	// PruneStates(int64, int64) error
	// // Close closes the connection with the database
	// Close() error
}

// dbStore wraps a db (github.com/tendermint/tm-db)
type dbStore struct {
	db dbm.DB
}

var _ Store = (*dbStore)(nil)

// NewStore creates the dbStore of the state pkg.
func NewStore(db dbm.DB) Store {
	return dbStore{db}
}

func (store dbStore) Load() (State, error) {
	return store.loadState(stateKey)
}

func (store dbStore) loadState(key []byte) (state State, err error) {
	buf, err := store.db.Get(key)
	if err != nil {
		return state, err
	}
	if len(buf) == 0 {
		return state, nil
	}

	sp := new(tmstate.State)

	err = proto.Unmarshal(buf, sp)
	if err != nil {
		// DATA HAS BEEN CORRUPTED OR THE SPEC HAS CHANGED
		tmos.Exit(fmt.Sprintf(`LoadState: Data has been corrupted or its spec has changed:
		%v\n`, err))
	}

	sm, err := FromProto(sp)
	if err != nil {
		return state, err
	}

	return *sm, nil
}
