package pack

import (
	"errors"
	"io"

	"filippo.io/age"
)

// EncryptStream encrypts src into dst using an age scrypt recipient derived
// from passphrase. An empty passphrase is rejected.
func EncryptStream(dst io.Writer, src io.Reader, passphrase string) error {
	if passphrase == "" {
		return errors.New("passphrase required")
	}
	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return err
	}
	w, err := age.Encrypt(dst, recipient)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, src); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}

// DecryptStream decrypts an age scrypt archive from src into dst. A wrong
// passphrase or corrupt header returns an error before any payload is copied,
// so callers can decrypt to a buffer and leave the destination untouched on
// failure.
func DecryptStream(dst io.Writer, src io.Reader, passphrase string) error {
	if passphrase == "" {
		return errors.New("passphrase required")
	}
	identity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return err
	}
	r, err := age.Decrypt(src, identity)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, r)
	return err
}
