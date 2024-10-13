package utils

import (
	"encoding/base64"
	"fmt"
)

// EncodeExpression encodes a dice roll expression to base64
func EncodeExpression(expression string) string {
	return base64.URLEncoding.EncodeToString([]byte(expression))
}

// DecodeExpression decodes a base64 encoded dice roll expression
func DecodeExpression(encoded string) (string, error) {
	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode expression: %v", err)
	}
	return string(decoded), nil
}
