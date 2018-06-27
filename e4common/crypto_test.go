package e4common

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestEncDec(t *testing.T) {

	ptLen := 1234

	key := make([]byte, KeyLen)
	ad := make([]byte, TimestampLen)
	pt := make([]byte, ptLen)

	rand.Read(key)
	rand.Read(ad)
	rand.Read(pt)

	ct, err := Encrypt(key, ad, pt)

	if err != nil {
		t.Fatalf("encryption failed: %s", err)
	}
	if len(ct) != len(pt)+TagLen {
		t.Fatalf("invalid ciphertext size: %d vs %d", len(ct), len(pt)+TagLen)
	}

	ptt, err := Decrypt(key, ad, ct)
	if err != nil {
		t.Fatalf("decryption failed: %s", err)
	}
	if len(pt) != len(ptt) {
		t.Fatalf("decrypted message has different length than original: %d vs %d", len(ptt), len(pt))
	}

	if !bytes.Equal(pt, ptt) {
		t.Fatalf("decrypted message different from the original")
	}
}


func TestProtectUnprotect(t *testing.T) {

	msgLen := 123

	key := make([]byte, KeyLen)
	msg := make([]byte, msgLen)

	rand.Read(key)
	rand.Read(msg)

	protected, err := Protect(msg, key)
	if err != nil {
		t.Fatalf("protect failed: %s", err)
	}

	unprotected, err := Unprotect(protected, key)
	if err != nil {
		t.Fatalf("unprotect failed: %s", err)
	}
	if !bytes.Equal(unprotected, msg) {
		t.Fatalf("unprotected message different from the original")
	}
}


