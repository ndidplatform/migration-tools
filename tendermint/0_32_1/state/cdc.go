package state

import (
	amino "github.com/tendermint/go-amino"

	"github.com/ndidplatform/migration-tools/tendermint/0_32_1/crypto"
	"github.com/ndidplatform/migration-tools/tendermint/0_32_1/crypto/ed25519"
	"github.com/ndidplatform/migration-tools/tendermint/0_32_1/crypto/multisig"
	"github.com/ndidplatform/migration-tools/tendermint/0_32_1/crypto/secp256k1"
)

var cdc = amino.NewCodec()

func init() {
	cdc.RegisterInterface((*crypto.PubKey)(nil), nil)
	cdc.RegisterConcrete(ed25519.PubKeyEd25519{},
		ed25519.PubKeyAminoName, nil)
	cdc.RegisterConcrete(secp256k1.PubKeySecp256k1{},
		secp256k1.PubKeyAminoName, nil)
	cdc.RegisterConcrete(multisig.PubKeyMultisigThreshold{},
		multisig.PubKeyMultisigThresholdAminoRoute, nil)

	cdc.RegisterInterface((*crypto.PrivKey)(nil), nil)
	cdc.RegisterConcrete(ed25519.PrivKeyEd25519{},
		ed25519.PrivKeyAminoName, nil)
	cdc.RegisterConcrete(secp256k1.PrivKeySecp256k1{},
		secp256k1.PrivKeyAminoName, nil)
}
