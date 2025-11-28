// Package validatorpk provides abstractions for handling validator public keys.
// It defines a generic PubKey structure that supports multiple cryptographic schemes
// (though currently focused on Secp256k1) and provides utilities for serialization,
// deserialization, and hex string conversion. This abstraction allows the consensus
// engine to work with public keys without needing to know the underlying curve details everywhere.

package validatorpk

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

const (
	// FakePassword is a constant string often used in testing or keystore placeholders
	// where a dummy password is required but security is not the primary concern.
	FakePassword = "fakepassword"
)

// PubKey represents a validator's public key.
// It decouples the key type from the raw bytes, allowing support for different
// signature schemes (e.g., Secp256k1, BLS) in the future.
type PubKey struct {
	// Type identifies the cryptographic curve or algorithm used (e.g., Secp256k1).
	Type uint8
	// Raw contains the actual public key bytes.
	Raw []byte
}

// Types defines the supported public key types constants.
// Currently, it only explicitly supports Secp256k1.
var Types = struct {
	Secp256k1 uint8
}{
	// Secp256k1 is the identifier for the standard Ethereum elliptic curve.
	// 0xc0 is an arbitrary byte value chosen to identify this type.
	Secp256k1: 0xc0,
}

// Empty checks if the public key is uninitialized or zeroed out.
// It returns true if both the Raw bytes are empty and the Type is 0.
func (pk PubKey) Empty() bool {
	return len(pk.Raw) == 0 && pk.Type == 0
}

// String returns the hexadecimal string representation of the public key, prefixed with "0x".
// It includes the Type byte prefix followed by the Raw bytes.
func (pk PubKey) String() string {
	return "0x" + common.Bytes2Hex(pk.Bytes())
}

// Bytes returns the flat byte slice representation of the public key.
// The format is [Type byte] + [Raw bytes...].
func (pk PubKey) Bytes() []byte {
	return append([]byte{pk.Type}, pk.Raw...)
}

// Copy creates a deep copy of the PubKey.
// This is important because the 'Raw' field is a slice (reference type),
// so a simple assignment would share the underlying memory.
func (pk PubKey) Copy() PubKey {
	return PubKey{
		Type: pk.Type,
		Raw:  common.CopyBytes(pk.Raw),
	}
}

// FromString parses a hex string (with or without "0x" prefix) into a PubKey.
// It first converts the hex string to bytes, then delegates to FromBytes.
func FromString(str string) (PubKey, error) {
	return FromBytes(common.FromHex(str))
}

// FromBytes reconstructs a PubKey from a flat byte slice.
// It expects the first byte to be the Type and the rest to be the Raw key.
// Returns an error if the slice is empty.
func FromBytes(b []byte) (PubKey, error) {
	if len(b) == 0 {
		return PubKey{}, errors.New("empty pubkey")
	}
	// b[0] is the Type, b[1:] is the Raw key data
	return PubKey{b[0], b[1:]}, nil
}

// MarshalText implements the encoding.TextMarshaler interface.
// This allows the PubKey to be automatically marshaled into a JSON string (as hex)
// when using standard Go JSON encoding.
func (pk *PubKey) MarshalText() ([]byte, error) {
	return []byte(pk.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
// This allows the PubKey to be automatically unmarshaled from a JSON string (hex)
// when using standard Go JSON decoding.
func (pk *PubKey) UnmarshalText(input []byte) error {
	res, err := FromString(string(input))
	if err != nil {
		return err
	}
	*pk = res
	return nil
}
