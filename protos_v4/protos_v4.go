package protos_v4

import (
	"github.com/golang/protobuf/proto"
)

func Unmarshal(b []byte, m proto.Message) error {
	return proto.Unmarshal([]byte(b), m)
}

func Marshal(m proto.Message) ([]byte, error) {
	return proto.Marshal(m)
}

func DeterministicMarshal(m proto.Message) ([]byte, error) {
	var b proto.Buffer
	b.SetDeterministic(true)
	if err := b.Marshal(m); err != nil {
		return nil, err
	}
	retBytes := b.Bytes()
	if retBytes == nil {
		retBytes = make([]byte, 0)
	}
	return retBytes, nil
}
