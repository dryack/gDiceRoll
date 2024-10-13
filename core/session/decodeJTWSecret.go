package session

import (
	"encoding/hex"
)

func decodeSecret(hexSecret string) ([]byte, error) {
	return hex.DecodeString(hexSecret)
}
