package encryption

import (
	"testing"
)

func TestEncryptGCM(t *testing.T) {
	cases := []struct {
		name, plain, key string
		sameOut, hasError bool
	} {
		{
			name: "should error on key < 32 bits",
			plain: "plain text",
			key: "01234567890",
			sameOut: false,
			hasError: true,
		},
		{
			name: "encrypt plain text",
			plain: "plain text",
			key: "01234567890123456789012345678901",
			sameOut: false,
			hasError: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			enc, err := EncryptGCM(tc.plain, tc.key)
			if tc.hasError && err == nil {
				t.Fatal("should have error")
			}
			if !tc.sameOut && string(enc) == tc.plain {
				t.Fatal("should not return plaintext")
			}
		})
	}
}

func TestDecryptGCM(t *testing.T) {
	getCypher := func(t *testing.T, plainText, key string) string {
		enc, err := EncryptGCM(plainText, key)
		if err != nil {
			t.Fatal("Failed to encrypt plain text")
		}
		return string(enc)
	}
	keys := []string{"01234567890123456789012345678901", "012345678901234567890123456789999"}
	plainText := "plain text"
	cypher := getCypher(t, plainText, keys[0])
	// should decipher using same key
	dec, err := DecryptGCM(cypher, keys[0])
	if string(dec) != plainText {
		t.Fatal("deciphered text not matched")
	}
	// should not decipher using different key
	dec, err = DecryptGCM(cypher, keys[1])
	if err == nil {
		t.Fatal("should error on differing keys")
	}
}
