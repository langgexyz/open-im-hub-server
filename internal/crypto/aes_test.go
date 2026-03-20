package crypto_test

import (
	"testing"

	"github.com/langgexyz/open-im-hub-server/internal/crypto"
)

func TestAESRoundTrip(t *testing.T) {
	key := "12345678901234567890123456789012" // 32 bytes
	plaintext := []byte(`{"node_id":"abc","node_private_key":"0xdeadbeef"}`)

	ciphertext, err := crypto.AESEncrypt([]byte(key), plaintext)
	if err != nil {
		t.Fatal(err)
	}
	got, err := crypto.AESDecrypt([]byte(key), ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(plaintext) {
		t.Fatalf("got %q, want %q", got, plaintext)
	}
}

func TestAESDecryptWrongKey(t *testing.T) {
	key := "12345678901234567890123456789012"
	plaintext := []byte("hello")
	ciphertext, _ := crypto.AESEncrypt([]byte(key), plaintext)

	wrongKey := "00000000000000000000000000000000"
	_, err := crypto.AESDecrypt([]byte(wrongKey), ciphertext)
	if err == nil {
		t.Fatal("expected error with wrong key")
	}
}
