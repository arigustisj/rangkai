package pack

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	var enc bytes.Buffer
	if err := EncryptStream(&enc, strings.NewReader("secret-payload"), "correct-pass"); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(enc.Bytes(), []byte("secret-payload")) {
		t.Fatal("ciphertext leaks plaintext")
	}
	var dec bytes.Buffer
	if err := DecryptStream(&dec, &enc, "correct-pass"); err != nil {
		t.Fatal(err)
	}
	if dec.String() != "secret-payload" {
		t.Fatalf("got %q", dec.String())
	}
}

func TestDecryptWrongPassphrase(t *testing.T) {
	var enc bytes.Buffer
	_ = EncryptStream(&enc, strings.NewReader("x"), "right")
	if err := DecryptStream(io.Discard, &enc, "wrong"); err == nil {
		t.Fatal("expected error")
	}
}

func TestEmptyPassphraseRejected(t *testing.T) {
	if err := EncryptStream(io.Discard, strings.NewReader("x"), ""); err == nil {
		t.Fatal("expected error")
	}
}
