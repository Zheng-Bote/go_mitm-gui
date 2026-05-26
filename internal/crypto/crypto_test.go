package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	password := "test-master-password-123"
	plaintext := []byte("Hello, mitm-gui! This is sensitive configuration data.")

	encrypted, err := Encrypt(password, plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Verify magic bytes.
	if string(encrypted[:4]) != MagicBytes {
		t.Fatalf("expected magic %q, got %q", MagicBytes, encrypted[:4])
	}

	// Verify minimum length.
	if len(encrypted) <= headerLen {
		t.Fatalf("encrypted data too short: %d bytes", len(encrypted))
	}

	// Decrypt.
	decrypted, err := Decrypt(password, encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("round-trip mismatch:\n  got:  %q\n  want: %q", decrypted, plaintext)
	}
}

func TestDecrypt_WrongPassword(t *testing.T) {
	password := "correct-password"
	plaintext := []byte("sensitive data")

	encrypted, err := Encrypt(password, plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt("wrong-password", encrypted)
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}

func TestDecrypt_InvalidMagic(t *testing.T) {
	// Must be >= headerLen+aes.BlockSize (48 bytes) to pass length check.
	data := make([]byte, 64)
	copy(data, "NOTM")
	_, err := Decrypt("any", data)
	if err != ErrInvalidMagic {
		t.Fatalf("expected ErrInvalidMagic, got %v", err)
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	_, err := Decrypt("any", []byte{})
	if err != ErrCiphertextTooShort {
		t.Fatalf("expected ErrCiphertextTooShort, got %v", err)
	}
}

func TestIsEncrypted(t *testing.T) {
	encrypted, err := Encrypt("pwd", []byte("data"))
	if err != nil {
		t.Fatal(err)
	}
	if !IsEncrypted(encrypted) {
		t.Fatal("expected IsEncrypted to return true")
	}
	if IsEncrypted([]byte("plain old data")) {
		t.Fatal("expected IsEncrypted to return false for plain data")
	}
}

func TestParseHeader(t *testing.T) {
	password := "test-password"
	plaintext := []byte("some config data")

	encrypted, err := Encrypt(password, plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	header, err := ParseHeader(encrypted)
	if err != nil {
		t.Fatalf("ParseHeader failed: %v", err)
	}

	if header.Magic != MagicBytes {
		t.Fatalf("expected magic %q, got %q", MagicBytes, header.Magic)
	}
	if len(header.Salt) != SaltLen {
		t.Fatalf("expected salt length %d, got %d", SaltLen, len(header.Salt))
	}
	if len(header.Nonce) != NonceLen {
		t.Fatalf("expected nonce length %d, got %d", NonceLen, len(header.Nonce))
	}
}

func TestEncrypt_UniqueSaltAndNonce(t *testing.T) {
	password := "same-password"
	data := []byte("same data")

	enc1, err := Encrypt(password, data)
	if err != nil {
		t.Fatal(err)
	}
	enc2, err := Encrypt(password, data)
	if err != nil {
		t.Fatal(err)
	}

	// Salt is at bytes 4-19, nonce at bytes 20-31.
	salt1 := enc1[4 : 4+SaltLen]
	salt2 := enc2[4 : 4+SaltLen]
	if bytes.Equal(salt1, salt2) {
		t.Fatal("expected different salts for two encryptions")
	}

	nonce1 := enc1[4+SaltLen : headerLen]
	nonce2 := enc2[4+SaltLen : headerLen]
	if bytes.Equal(nonce1, nonce2) {
		t.Fatal("expected different nonces for two encryptions")
	}
}

func TestBinarySize(t *testing.T) {
	for _, tc := range []struct {
		plainLen int
		want     int64
	}{
		{0, int64(headerLen + 0 + 16)},
		{100, int64(headerLen + 100 + 16)},
		{1024, int64(headerLen + 1024 + 16)},
	} {
		got := BinarySize(tc.plainLen)
		if got != tc.want {
			t.Errorf("BinarySize(%d) = %d, want %d", tc.plainLen, got, tc.want)
		}
	}
}

func TestDecrypt_NilData(t *testing.T) {
	_, err := Decrypt("pwd", nil)
	if err != ErrCiphertextTooShort {
		t.Fatalf("expected ErrCiphertextTooShort for nil data, got %v", err)
	}
}
