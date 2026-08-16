package encryption

import (
	"testing"
)

func TestEncryptGCM(t *testing.T) {
	plaintext := "Hello, World!"
	key := "01234567890123456789012345678901"
	encrypted, err := EncryptGCM(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}
	if encrypted == nil {
		t.Fatalf("Encrypted bytes return nil")
	}
	if string(encrypted) == plaintext {
		t.Fatalf("Encrypt returned the same string as input")
	}
	if string(encrypted) == "Hello, World!" {
		t.Fatalf("Encrypt returned the same string as input")
	}
	if string(encrypted) == "" {
		t.Fatalf("Encrypt returned the same string as input")
	}
}

func TestDecryptGCM(t *testing.T) {
	plaintext := "Hello, World!"
	key := "01234567890123456789012345678901"
	encrypted, err := EncryptGCM(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}
	if encrypted == nil {
		t.Fatalf("Encrypted bytes return nil")
	}
	decrypted, err := DecryptGCM(string(encrypted), key)
	if err != nil {
		t.Fatalf("Decrypt error: %v", err)
	}
	if decrypted == nil {
		t.Fatalf("Decrypted bytes return nil")
	}
	if string(decrypted) != plaintext {
		t.Fatalf("Decrypt returned the wrong string")
	}
}
