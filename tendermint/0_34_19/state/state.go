package state

import (
	"errors"
	"time"

	tmstate "github.com/ndidplatform/migration-tools/tendermint/0_34_19/proto/tendermint/state"
	tmversion "github.com/ndidplatform/migration-tools/tendermint/0_34_19/proto/tendermint/version"
	"github.com/ndidplatform/migration-tools/tendermint/0_34_19/types"
	"github.com/ndidplatform/migration-tools/tendermint/0_34_19/version"
)

// database keys
var (
	stateKey = []byte("stateKey")
)

//-----------------------------------------------------------------------------

// InitStateVersion sets the Consensus.Block and Software versions,
// but leaves the Consensus.App version blank.
// The Consensus.App version will be set during the Handshake, once
// we hear from the app what protocol version it is running.
var InitStateVersion = tmstate.Version{
	Consensus: tmversion.Consensus{
		Block: version.BlockProtocol,
		App:   0,
	},
	Software: version.TMCoreSemVer,
}

//-----------------------------------------------------------------------------

// State is a short description of the latest committed block of the Tendermint consensus.
// It keeps all information necessary to validate new blocks,
// including the last validator set and the consensus params.
// All fields are exposed so the struct can be easily serialized,
// but none of them should be mutated directly.
// Instead, use state.Copy() or state.NextState(...).
// NOTE: not goroutine-safe.
type State struct {
	Version tmstate.Version

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
	// Changes returned by EndBlock and updated after Commit.
	// ConsensusParams                  tmproto.ConsensusParams
	// LastHeightConsensusParamsChanged int64

	// Merkle root of the results from executing prev block
	LastResultsHash []byte

	// the latest AppHash we've received from calling abci.Commit()
	AppHash []byte
}

// FromProto takes a state proto message & returns the local state type
func FromProto(pb *tmstate.State) (*State, error) { //nolint:golint
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
	// state.ConsensusParams = pb.ConsensusParams
	// state.LastHeightConsensusParamsChanged = pb.LastHeightConsensusParamsChanged
	state.LastResultsHash = pb.LastResultsHash
	state.AppHash = pb.AppHash

	return state, nil
}
