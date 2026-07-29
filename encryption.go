package gateman

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// EncryptGCM encrypts the plaintext using AES-GCM.
//
// Example:
//	plaintext := []byte("secret message")
//	key := []byte("01234567890123456789012345678901") // 32 bytes for AES-256
//	
//	encrypted, err := gateman.EncryptGCM(plaintext, key)
//	if err != nil {
//		// handle error
//	}
//	
//	fmt.Println("Encrypted:", encrypted)
//
func EncryptGCM(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// DecryptGCM decrypts the ciphertext using AES-GCM.
//
// Example:
//	ciphertext := []byte{...} // encrypted data from EncryptGCM
//	key := []byte("01234567890123456789012345678901")
//	
//	decrypted, err := gateman.DecryptGCM(ciphertext, key)
//	if err != nil {
//		// handle error (e.g., wrong key, corrupted data)
//	}
//	
//	fmt.Println("Decrypted:", string(decrypted))
//
func DecryptGCM(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
