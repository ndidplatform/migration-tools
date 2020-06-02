package multisig

import (
	"github.com/ndidplatform/migration-tools/tendermint/0_32_1/crypto"
	"github.com/ndidplatform/migration-tools/tendermint/0_32_1/crypto/ed25519"
	"github.com/ndidplatform/migration-tools/tendermint/0_32_1/crypto/secp256k1"
	amino "github.com/tendermint/go-amino"
)

// TODO: Figure out API for others to either add their own pubkey types, or
// to make verify / marshal accept a cdc.
const (
	PubKeyMultisigThresholdAminoRoute = "tendermint/PubKeyMultisigThreshold"
)

var cdc = amino.NewCodec()

func init() {
	cdc.RegisterInterface((*crypto.PubKey)(nil), nil)
	cdc.RegisterConcrete(PubKeyMultisigThreshold{},
		PubKeyMultisigThresholdAminoRoute, nil)
	cdc.RegisterConcrete(ed25519.PubKeyEd25519{},
		ed25519.PubKeyAminoName, nil)
	cdc.RegisterConcrete(secp256k1.PubKeySecp256k1{},
		secp256k1.PubKeyAminoName, nil)
}
