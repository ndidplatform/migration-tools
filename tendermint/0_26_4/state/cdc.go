package state

import (
	amino "github.com/tendermint/go-amino"

	"github.com/ndidplatform/migration-tools/tendermint/0_26_4/crypto"
	"github.com/ndidplatform/migration-tools/tendermint/0_26_4/crypto/ed25519"
	"github.com/ndidplatform/migration-tools/tendermint/0_26_4/crypto/multisig"
	"github.com/ndidplatform/migration-tools/tendermint/0_26_4/crypto/secp256k1"
)

var cdc = amino.NewCodec()

func init() {
	cdc.RegisterInterface((*crypto.PubKey)(nil), nil)
	cdc.RegisterConcrete(ed25519.PubKeyEd25519{},
		ed25519.PubKeyAminoRoute, nil)
	cdc.RegisterConcrete(secp256k1.PubKeySecp256k1{},
		secp256k1.PubKeyAminoRoute, nil)
	cdc.RegisterConcrete(multisig.PubKeyMultisigThreshold{},
		multisig.PubKeyMultisigThresholdAminoRoute, nil)

	cdc.RegisterInterface((*crypto.PrivKey)(nil), nil)
	cdc.RegisterConcrete(ed25519.PrivKeyEd25519{},
		ed25519.PrivKeyAminoRoute, nil)
	cdc.RegisterConcrete(secp256k1.PrivKeySecp256k1{},
		secp256k1.PrivKeyAminoRoute, nil)
}
