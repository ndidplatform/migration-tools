package batch

import (
	"github.com/ndidplatform/migration-tools/tendermint/0_38_6/crypto"
	"github.com/ndidplatform/migration-tools/tendermint/0_38_6/crypto/ed25519"
	"github.com/ndidplatform/migration-tools/tendermint/0_38_6/crypto/sr25519"
)

// CreateBatchVerifier checks if a key type implements the batch verifier interface.
// Currently only ed25519 & sr25519 supports batch verification.
func CreateBatchVerifier(pk crypto.PubKey) (crypto.BatchVerifier, bool) {
	switch pk.Type() {
	case ed25519.KeyType:
		return ed25519.NewBatchVerifier(), true
	case sr25519.KeyType:
		return sr25519.NewBatchVerifier(), true
	}

	// case where the key does not support batch verification
	return nil, false
}

// SupportsBatchVerifier checks if a key type implements the batch verifier
// interface.
func SupportsBatchVerifier(pk crypto.PubKey) bool {
	if pk == nil {
		return false
	}
	switch pk.Type() {
	case ed25519.KeyType, sr25519.KeyType:
		return true
	}

	return false
}
