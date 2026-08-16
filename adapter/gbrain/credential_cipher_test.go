package gbrain

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestCredentialCipherBindsCiphertextToBrainAndClient(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
	cipher, err := newCredentialCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := cipher.encrypt("brain-id", "client-id", "client-secret")
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := cipher.decrypt("brain-id", "client-id", ciphertext)
	if err != nil || plaintext != "client-secret" {
		t.Fatalf("decrypt = %q, %v", plaintext, err)
	}
	if _, err := cipher.decrypt("other-brain", "client-id", ciphertext); err == nil {
		t.Fatal("ciphertext decrypted for a different brain")
	}
	if _, err := cipher.decrypt("brain-id", "other-client", ciphertext); err == nil {
		t.Fatal("ciphertext decrypted for a different client")
	}
}
