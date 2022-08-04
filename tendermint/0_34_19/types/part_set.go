package types

import (
	"errors"
	"fmt"

	"github.com/ndidplatform/migration-tools/tendermint/0_34_19/crypto/merkle"
	"github.com/ndidplatform/migration-tools/tendermint/0_34_19/libs/bits"
	tmbytes "github.com/ndidplatform/migration-tools/tendermint/0_34_19/libs/bytes"
	tmsync "github.com/ndidplatform/migration-tools/tendermint/0_34_19/libs/sync"
	tmproto "github.com/ndidplatform/migration-tools/tendermint/0_34_19/proto/tendermint/types"
)

type Part struct {
	Index uint32           `json:"index"`
	Bytes tmbytes.HexBytes `json:"bytes"`
	Proof merkle.Proof     `json:"proof"`
}

type PartSetHeader struct {
	Total uint32           `json:"total"`
	Hash  tmbytes.HexBytes `json:"hash"`
}

// ValidateBasic performs basic validation.
func (psh PartSetHeader) ValidateBasic() error {
	// Hash can be empty in case of POLBlockID.PartSetHeader in Proposal.
	if err := ValidateHash(psh.Hash); err != nil {
		return fmt.Errorf("wrong Hash: %w", err)
	}
	return nil
}

// FromProto sets a protobuf PartSetHeader to the given pointer
func PartSetHeaderFromProto(ppsh *tmproto.PartSetHeader) (*PartSetHeader, error) {
	if ppsh == nil {
		return nil, errors.New("nil PartSetHeader")
	}
	psh := new(PartSetHeader)
	psh.Total = ppsh.Total
	psh.Hash = ppsh.Hash

	return psh, psh.ValidateBasic()
}

//-------------------------------------

type PartSet struct {
	total uint32
	hash  []byte

	mtx           tmsync.Mutex
	parts         []*Part
	partsBitArray *bits.BitArray
	count         uint32
	// a count of the total size (in bytes). Used to ensure that the
	// part set doesn't exceed the maximum block bytes
	byteSize int64
}
