package cache

import (
	"bytes"
	"encoding/binary"
)

// BoolToBytes converts a boolean into a slice of bytes.
func BoolToBytes(b bool) []byte {
	if b {
		return []byte("1")
	}
	return []byte("0")
}

// BytesToBool converts a slice of bytes into a boolean value.
func BytesToBool(b []byte) bool {
	return bytes.Compare([]byte("1"), b) == 0
}

// BytesToInt converts a slice of bytes into an integer.
func BytesToInt(b []byte) int64 {
	i, _ := binary.Varint(b)
	return i
}

// IntToBytes converts an integer into a slice of bytes
func IntToBytes(i int64) []byte {
	b := make([]byte, 1)
	binary.PutVarint(b, i)
	return b
}
