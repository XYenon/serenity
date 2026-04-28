package subscription

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/sagernet/serenity/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaybeDecryptSubscriptionContent(t *testing.T) {
	decrypt, err := NewDecryptOptions(&option.SubscriptionDecryptOptions{
		Key: "ZpGhHIPrDuTbvHSuk+2wyU2AA16jsngz",
	})
	require.NoError(t, err)

	header := make(http.Header)
	header.Set(defaultSubscriptionDecryptResponseHeader, defaultSubscriptionDecryptResponseHeaderValue)
	plaintext := []byte("mixed-port: 7890\nallow-lan: true\n")
	encrypted := encryptSubscriptionContent(t, plaintext, decrypt.key)

	decrypted, err := maybeDecryptSubscriptionContent(header, encrypted, decrypt)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestMaybeDecryptSubscriptionContentRequiresKey(t *testing.T) {
	header := make(http.Header)
	header.Set(defaultSubscriptionDecryptResponseHeader, defaultSubscriptionDecryptResponseHeaderValue)

	_, err := maybeDecryptSubscriptionContent(header, []byte("ignored"), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subscription.decrypt.key")
}

func TestMaybeDecryptSubscriptionContentSkipsWhenHeaderMissing(t *testing.T) {
	decrypt, err := NewDecryptOptions(&option.SubscriptionDecryptOptions{
		Key: "ZpGhHIPrDuTbvHSuk+2wyU2AA16jsngz",
	})
	require.NoError(t, err)

	original := []byte("plain subscription")
	content, err := maybeDecryptSubscriptionContent(make(http.Header), original, decrypt)
	require.NoError(t, err)
	assert.Equal(t, original, content)
}

func TestMaybeDecryptSubscriptionContentCustomHeader(t *testing.T) {
	decrypt, err := NewDecryptOptions(&option.SubscriptionDecryptOptions{
		Key:                 "ZpGhHIPrDuTbvHSuk+2wyU2AA16jsngz",
		ResponseHeader:      "x-custom-encrypted",
		ResponseHeaderValue: "yes",
	})
	require.NoError(t, err)

	header := make(http.Header)
	header.Set("x-custom-encrypted", "yes")
	plaintext := []byte("mixed-port: 7890\n")
	encrypted := encryptSubscriptionContent(t, plaintext, decrypt.key)

	decrypted, err := maybeDecryptSubscriptionContent(header, encrypted, decrypt)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestNewDecryptOptionsInvalidKeyLength(t *testing.T) {
	_, err := NewDecryptOptions(&option.SubscriptionDecryptOptions{Key: "short"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}

func encryptSubscriptionContent(t *testing.T, plaintext []byte, key []byte) []byte {
	t.Helper()
	block, err := aes.NewCipher(key)
	require.NoError(t, err)
	iv := []byte("1234567890abcdef")
	padded := pkcs7Pad(plaintext, aes.BlockSize)
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)
	blob := append(append([]byte{}, iv...), ciphertext...)
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(blob)))
	base64.StdEncoding.Encode(encoded, blob)
	return encoded
}

func pkcs7Pad(content []byte, blockSize int) []byte {
	padding := blockSize - len(content)%blockSize
	return append(append([]byte{}, content...), bytes.Repeat([]byte{byte(padding)}, padding)...)
}
