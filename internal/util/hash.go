package util

import (
	"crypto/sha256"
	"encoding/hex"
)

func SHA256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func SHA256Tagged(b []byte) string {
	return "sha256:" + SHA256Hex(b)
}
