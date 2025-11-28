// Public Key Test file contains unit tests for the validatorpk package.
// It verifies the serialization, deserialization, and manipulation logic for validator public keys,
// ensuring that keys can be correctly converted between their binary, hex string, and object representations.
package validatorpk

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestFromString verifies that a hexadecimal string (with or without 0x prefix)
// can be correctly parsed into a PubKey structure.
func TestFromString(t *testing.T) {
	require := require.New(t)

	// Define the expected PubKey object for the test cases.
	// Type is 0xc0 (Secp256k1), followed by the raw public key bytes.
	exp := PubKey{
		Type: Types.Secp256k1,
		Raw:  common.FromHex("45b86101f804f3f4f2012ef31fff807e87de579a3faa7947d1b487a810e35dc2c3b6071ac465046634b5f4a8e09bf8e1f2e7eccb699356b9e6fd496ca4b1677d1"),
	}

	// Case 1: Valid hex string without "0x" prefix.
	{
		got, err := FromString("c0045b86101f804f3f4f2012ef31fff807e87de579a3faa7947d1b487a810e35dc2c3b6071ac465046634b5f4a8e09bf8e1f2e7eccb699356b9e6fd496ca4b1677d1")
		require.NoError(err)
		require.Equal(exp, got)
	}

	// Case 2: Valid hex string with "0x" prefix.
	{
		got, err := FromString("0xc0045b86101f804f3f4f2012ef31fff807e87de579a3faa7947d1b487a810e35dc2c3b6071ac465046634b5f4a8e09bf8e1f2e7eccb699356b9e6fd496ca4b1677d1")
		require.NoError(err)
		require.Equal(exp, got)
	}

	// Case 3: Empty string should return an error.
	{
		_, err := FromString("")
		require.Error(err)
	}

	// Case 4: "0x" only (empty bytes) should return an error.
	{
		_, err := FromString("0x")
		require.Error(err)
	}

	// Case 5: Invalid hex characters should return an error.
	{
		_, err := FromString("-")
		require.Error(err)
	}
}

// TestString verifies that a PubKey object is correctly formatted as a hexadecimal string
// prefixed with "0x".
func TestString(t *testing.T) {
	require := require.New(t)
	pk := PubKey{
		Type: Types.Secp256k1,
		Raw:  common.FromHex("45b86101f804f3f4f2012ef31fff807e87de579a3faa7947d1b487a810e35dc2c3b6071ac465046634b5f4a8e09bf8e1f2e7eccb699356b9e6fd496ca4b1677d1"),
	}
	// The expected string starts with 0x, then the type byte (c0), then the raw bytes.
	require.Equal("0xc0045b86101f804f3f4f2012ef31fff807e87de579a3faa7947d1b487a810e35dc2c3b6071ac465046634b5f4a8e09bf8e1f2e7eccb699356b9e6fd496ca4b1677d1", pk.String())
}

// TestEmpty checks the behavior of the Empty() method.
func TestEmpty(t *testing.T) {
	require := require.New(t)

	// Case 1: A zero-value PubKey should be considered empty.
	emptyPk := PubKey{}
	require.True(emptyPk.Empty(), "Zero value PubKey should be empty")

	// Case 2: A populated PubKey should not be empty.
	validPk := PubKey{
		Type: Types.Secp256k1,
		Raw:  []byte{0x01},
	}
	require.False(validPk.Empty(), "Populated PubKey should not be empty")
}

// TestBytes verifies the conversion of PubKey to a flat byte slice.
func TestBytes(t *testing.T) {
	require := require.New(t)

	pk := PubKey{
		Type: 0x01,
		Raw:  []byte{0x02, 0x03},
	}

	// Expect concatenation of [Type] + [Raw...]
	expected := []byte{0x01, 0x02, 0x03}
	require.Equal(expected, pk.Bytes())
}

// TestCopy verifies that the Copy() method creates a deep copy of the PubKey.
func TestCopy(t *testing.T) {
	require := require.New(t)

	original := PubKey{
		Type: 0x01,
		Raw:  []byte{0xAA, 0xBB},
	}

	// Create a copy
	copyPk := original.Copy()

	// They should be equal initially
	require.Equal(original, copyPk)

	// Modify the underlying slice of the copy
	copyPk.Raw[0] = 0xFF

	// The original should remain unchanged (proving it was a deep copy)
	require.Equal(uint8(0xAA), original.Raw[0], "Original PubKey was modified by copy")
	require.NotEqual(original, copyPk)
}

// TestFromBytes verifies parsing a raw byte slice into a PubKey.
func TestFromBytes(t *testing.T) {
	require := require.New(t)

	// Case 1: Valid bytes (Type + Raw)
	input := []byte{0xc0, 0x01, 0x02}
	pk, err := FromBytes(input)
	require.NoError(err)
	require.Equal(uint8(0xc0), pk.Type)
	require.Equal([]byte{0x01, 0x02}, pk.Raw)

	// Case 2: Empty bytes should return error
	_, err = FromBytes([]byte{})
	require.Error(err)
}

// TestMarshalUnmarshal verifies JSON encoding and decoding via MarshalText/UnmarshalText.
func TestMarshalUnmarshal(t *testing.T) {
	require := require.New(t)

	original := PubKey{
		Type: Types.Secp256k1,
		Raw:  []byte{0xAA, 0xBB, 0xCC},
	}

	// Marshal to JSON
	data, err := json.Marshal(&original)
	require.NoError(err)

	// The JSON string should be the quoted hex string
	expectedJson := `"` + original.String() + `"`
	require.Equal(expectedJson, string(data))

	// Unmarshal back to struct
	var decoded PubKey
	err = json.Unmarshal(data, &decoded)
	require.NoError(err)

	require.Equal(original, decoded)
}
