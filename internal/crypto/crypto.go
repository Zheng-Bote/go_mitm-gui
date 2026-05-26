// Package crypto provides AES-256-GCM encryption with Argon2id key derivation.
//
// File format:
//
//	[0-3]   Magic bytes "MITM" (4 bytes)
//	[4-19]  Salt (16 bytes)
//	[20-31] Nonce (12 bytes)
//	[32+]   AES-256-GCM ciphertext || 16-byte GCM authentication tag
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	// MagicBytes is the 4-byte file header that identifies an encrypted mitm-gui file.
	MagicBytes = "MITM"

	// SaltLen is the length of the Argon2id salt in bytes.
	SaltLen = 16
	// NonceLen is the length of the AES-GCM nonce in bytes (12 is recommended for GCM).
	NonceLen = 12
	// KeyLen is the length of the derived AES key in bytes (32 bytes = AES-256).
	KeyLen = 32

	// Argon2idTime is the time parameter for Argon2id.
	Argon2idTime = 3
	// Argon2idMemory is the memory parameter for Argon2id in KiB (64 MiB).
	Argon2idMemory = 64 * 1024
	// Argon2idThreads is the parallelism parameter for Argon2id.
	Argon2idThreads = 4

	// headerLen is the total length of the metadata prefix.
	headerLen = 4 + SaltLen + NonceLen // 32 bytes
)

var (
	ErrInvalidMagic   = errors.New("crypto: invalid magic bytes — not a mitm-gui encrypted file")
	ErrCiphertextTooShort = errors.New("crypto: ciphertext too short")
	ErrDecryptFailed      = errors.New("crypto: decryption failed — wrong password or corrupted data")
)

// deriveKey derives a 32-byte AES-256 key from the given password and salt
// using Argon2id.
func deriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, Argon2idTime, Argon2idMemory, Argon2idThreads, KeyLen)
}

// Encrypt encrypts plaintext using AES-256-GCM with an Argon2id-derived key.
//
// The returned byte slice has the following layout:
//
//	[0-3]   "MITM" magic
//	[4-19]  random salt (16 bytes)
//	[20-31] random nonce (12 bytes)
//	[32+]   ciphertext || GCM tag (16 bytes)
//
// A new random salt and nonce are generated for every call.
func Encrypt(password string, plaintext []byte) ([]byte, error) {
	// Generate random salt.
	salt := make([]byte, SaltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	// Generate random nonce.
	nonce := make([]byte, NonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Derive key.
	key := deriveKey(password, salt)

	// Create AES cipher.
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create GCM.
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Encrypt (Seal prepends the nonce to the ciphertext, but we manage nonce
	// separately for our file format).
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Assemble: magic (4) + salt (16) + nonce (12) + ciphertext (plaintext + 16 tag).
	buf := make([]byte, headerLen+len(ciphertext))
	copy(buf[0:4], MagicBytes)
	copy(buf[4:4+SaltLen], salt)
	copy(buf[4+SaltLen:headerLen], nonce)
	copy(buf[headerLen:], ciphertext)

	return buf, nil
}

// Decrypt decrypts data produced by Encrypt.
//
// It expects the file format:
//
//	[0-3]   "MITM" magic
//	[4-19]  salt (16 bytes)
//	[20-31] nonce (12 bytes)
//	[32+]   ciphertext || GCM tag (16 bytes)
//
// Returns the original plaintext.
func Decrypt(password string, data []byte) ([]byte, error) {
	if len(data) < headerLen+aes.BlockSize {
		return nil, ErrCiphertextTooShort
	}

	// Validate magic bytes.
	if string(data[:4]) != MagicBytes {
		return nil, ErrInvalidMagic
	}

	// Extract salt and nonce.
	salt := data[4 : 4+SaltLen]
	nonce := data[4+SaltLen : headerLen]
	ciphertext := data[headerLen:]

	// Derive key.
	key := deriveKey(password, salt)

	// Create AES cipher.
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create GCM.
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Decrypt (Open verifies the GCM authentication tag).
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.Join(ErrDecryptFailed, err)
	}

	return plaintext, nil
}

// IsEncrypted reports whether the data begins with the magic bytes.
func IsEncrypted(data []byte) bool {
	return len(data) >= 4 && string(data[:4]) == MagicBytes
}

// EncryptFile encrypts a plaintext file read as raw bytes and returns
// the encrypted payload. This is a convenience wrapper for Encrypt.
func EncryptFile(password string, plaintext []byte) ([]byte, error) {
	return Encrypt(password, plaintext)
}

// FileHeader returns the magic, salt, and nonce from encrypted data
// without performing decryption. Useful for identifying files.
type FileHeader struct {
	Magic string
	Salt  []byte
	Nonce []byte
}

// ParseHeader extracts the header from encrypted data without decrypting.
func ParseHeader(data []byte) (*FileHeader, error) {
	if len(data) < headerLen {
		return nil, ErrCiphertextTooShort
	}
	if string(data[:4]) != MagicBytes {
		return nil, ErrInvalidMagic
	}
	salt := make([]byte, SaltLen)
	nonce := make([]byte, NonceLen)
	copy(salt, data[4:4+SaltLen])
	copy(nonce, data[4+SaltLen:headerLen])
	return &FileHeader{
		Magic: MagicBytes,
		Salt:  salt,
		Nonce: nonce,
	}, nil
}

// BinarySize returns the expected size of the encrypted payload given a
// plaintext of size n. Useful for buffer allocation.
func BinarySize(plaintextLen int) int64 {
	return int64(headerLen + plaintextLen + 16) // +16 for GCM tag
}
