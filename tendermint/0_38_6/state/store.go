package state

import (
	"fmt"

	"github.com/cosmos/gogoproto/proto"

	dbm "github.com/cometbft/cometbft-db"

	cmtos "github.com/ndidplatform/migration-tools/tendermint/0_38_6/libs/os"
	cmtstate "github.com/ndidplatform/migration-tools/tendermint/0_38_6/proto/tendermint/state"
)

const (
	// persist validators every valSetCheckpointInterval blocks to avoid
	// LoadValidators taking too much time.
	// https://github.com/tendermint/tendermint/pull/3438
	// 100000 results in ~ 100ms to get 100 validators (see BenchmarkLoadValidators)
	valSetCheckpointInterval = 100000
)

//------------------------------------------------------------------------

func calcValidatorsKey(height int64) []byte {
	return []byte(fmt.Sprintf("validatorsKey:%v", height))
}

func calcConsensusParamsKey(height int64) []byte {
	return []byte(fmt.Sprintf("consensusParamsKey:%v", height))
}

func calcABCIResponsesKey(height int64) []byte {
	return []byte(fmt.Sprintf("abciResponsesKey:%v", height))
}

//----------------------

var lastABCIResponseKey = []byte("lastABCIResponseKey")
var offlineStateSyncHeight = []byte("offlineStateSyncHeightKey")

//go:generate ../scripts/mockery_generate.sh Store

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
	// // LoadFinalizeBlockResponse loads the abciResponse for a given height
	// LoadFinalizeBlockResponse(int64) (*abci.ResponseFinalizeBlock, error)
	// // LoadLastABCIResponse loads the last abciResponse for a given height
	// LoadLastFinalizeBlockResponse(int64) (*abci.ResponseFinalizeBlock, error)
	// // LoadConsensusParams loads the consensus params for a given height
	// LoadConsensusParams(int64) (types.ConsensusParams, error)
	// // Save overwrites the previous state with the updated one
	// Save(State) error
	// // SaveFinalizeBlockResponse saves ABCIResponses for a given height
	// SaveFinalizeBlockResponse(int64, *abci.ResponseFinalizeBlock) error
	// // Bootstrap is used for bootstrapping state when not starting from a initial height.
	// Bootstrap(State) error
	// // PruneStates takes the height from which to start pruning and which height stop at
	// PruneStates(int64, int64, int64) error
	// // Saves the height at which the store is bootstrapped after out of band statesync
	// SetOfflineStateSyncHeight(height int64) error
	// // Gets the height at which the store is bootstrapped after out of band statesync
	// GetOfflineStateSyncHeight() (int64, error)
	// // Close closes the connection with the database
	// Close() error
}

// dbStore wraps a db (github.com/cometbft/cometbft-db)
type dbStore struct {
	db dbm.DB

	StoreOptions
}

type StoreOptions struct {
	// DiscardABCIResponses determines whether or not the store
	// retains all ABCIResponses. If DiscardABCIResponses is enabled,
	// the store will maintain only the response object from the latest
	// height.
	DiscardABCIResponses bool
}

var _ Store = (*dbStore)(nil)

// NewStore creates the dbStore of the state pkg.
func NewStore(db dbm.DB, options StoreOptions) Store {
	return dbStore{db, options}
}

// LoadState loads the State from the database.
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

	sp := new(cmtstate.State)

	err = proto.Unmarshal(buf, sp)
	if err != nil {
		// DATA HAS BEEN CORRUPTED OR THE SPEC HAS CHANGED
		cmtos.Exit(fmt.Sprintf(`LoadState: Data has been corrupted or its spec has changed:
		%v\n`, err))
	}

	sm, err := FromProto(sp)
	if err != nil {
		return state, err
	}
	return *sm, nil
}
