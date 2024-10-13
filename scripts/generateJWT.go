package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func generateJWTSecret(bytes int) (string, error) {
	// Create a byte slice to hold the random bytes
	secret := make([]byte, bytes)

	// Fill the slice with random bytes
	_, err := rand.Read(secret)
	if err != nil {
		return "", fmt.Errorf("error generating random bytes: %v", err)
	}

	// Convert the byte slice to a hexadecimal string
	return hex.EncodeToString(secret), nil
}

func main() {
	// Generate a 32-byte (256-bit) secret
	accessSecret, err := generateJWTSecret(32)
	if err != nil {
		fmt.Printf("Error generating access secret: %v\n", err)
		return
	}

	refreshSecret, err := generateJWTSecret(32)
	if err != nil {
		fmt.Printf("Error generating refresh secret: %v\n", err)
		return
	}

	fmt.Printf("Access Secret: %s\n", accessSecret)
	fmt.Printf("Refresh Secret: %s\n", refreshSecret)
}
