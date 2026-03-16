package state

import (
	"errors"
	"time"

	cmtstate "github.com/ndidplatform/migration-tools/tendermint/0_38_6/proto/tendermint/state"
	"github.com/ndidplatform/migration-tools/tendermint/0_38_6/types"
)

// database keys
var (
	stateKey = []byte("stateKey")
)

//-----------------------------------------------------------------------------

// State is a short description of the latest committed block of the consensus protocol.
// It keeps all information necessary to validate new blocks,
// including the last validator set and the consensus params.
// All fields are exposed so the struct can be easily serialized,
// but none of them should be mutated directly.
// Instead, use state.Copy() or state.NextState(...).
// NOTE: not goroutine-safe.
type State struct {
	Version cmtstate.Version

	// immutable
	ChainID       string
	InitialHeight int64 // should be 1, not 0, when starting from height 1

	// LastBlockHeight=0 at genesis (ie. block(H=0) does not exist)
	LastBlockHeight int64
	LastBlockID     types.BlockID
	LastBlockTime   time.Time

	// LastValidators is used to validate block.LastCommit.
	// Validators are persisted to the database separately every time they change,
	// so we can query for historical validator sets.
	// Note that if s.LastBlockHeight causes a valset change,
	// we set s.LastHeightValidatorsChanged = s.LastBlockHeight + 1 + 1
	// Extra +1 due to nextValSet delay.
	// NextValidators              *types.ValidatorSet
	// Validators                  *types.ValidatorSet
	// LastValidators              *types.ValidatorSet
	// LastHeightValidatorsChanged int64

	// Consensus parameters used for validating blocks.
	// Changes returned by FinalizeBlock and updated after Commit.
	// ConsensusParams                  types.ConsensusParams
	// LastHeightConsensusParamsChanged int64

	// Merkle root of the results from executing prev block
	LastResultsHash []byte

	// the latest AppHash we've received from calling abci.Commit()
	AppHash []byte
}

// FromProto takes a state proto message & returns the local state type
func FromProto(pb *cmtstate.State) (*State, error) { //nolint:golint
	if pb == nil {
		return nil, errors.New("nil State")
	}

	state := new(State)

	state.Version = pb.Version
	state.ChainID = pb.ChainID
	state.InitialHeight = pb.InitialHeight

	bi, err := types.BlockIDFromProto(&pb.LastBlockID)
	if err != nil {
		return nil, err
	}
	state.LastBlockID = *bi
	state.LastBlockHeight = pb.LastBlockHeight
	state.LastBlockTime = pb.LastBlockTime

	// vals, err := types.ValidatorSetFromProto(pb.Validators)
	// if err != nil {
	// 	return nil, err
	// }
	// state.Validators = vals

	// nVals, err := types.ValidatorSetFromProto(pb.NextValidators)
	// if err != nil {
	// 	return nil, err
	// }
	// state.NextValidators = nVals

	// if state.LastBlockHeight >= 1 { // At Block 1 LastValidators is nil
	// 	lVals, err := types.ValidatorSetFromProto(pb.LastValidators)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	state.LastValidators = lVals
	// } else {
	// 	state.LastValidators = types.NewValidatorSet(nil)
	// }

	// state.LastHeightValidatorsChanged = pb.LastHeightValidatorsChanged
	// state.ConsensusParams = types.ConsensusParamsFromProto(pb.ConsensusParams)
	// state.LastHeightConsensusParamsChanged = pb.LastHeightConsensusParamsChanged
	state.LastResultsHash = pb.LastResultsHash
	state.AppHash = pb.AppHash

	return state, nil
}
