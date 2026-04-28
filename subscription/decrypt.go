package subscription

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/sagernet/serenity/option"
	E "github.com/sagernet/sing/common/exceptions"
)

const (
	defaultSubscriptionDecryptResponseHeader      = "x-flclash-encrypted"
	defaultSubscriptionDecryptResponseHeaderValue = "1"
)

type DecryptOptions struct {
	key                 []byte
	responseHeader      string
	responseHeaderValue string
}

func NewDecryptOptions(raw *option.SubscriptionDecryptOptions) (*DecryptOptions, error) {
	if raw == nil {
		return nil, nil
	}
	if raw.Key == "" {
		return nil, E.New("missing key")
	}
	key := []byte(raw.Key)
	if len(key) != 32 {
		return nil, E.New("key must be 32 bytes")
	}
	responseHeader := strings.TrimSpace(raw.ResponseHeader)
	if responseHeader == "" {
		responseHeader = defaultSubscriptionDecryptResponseHeader
	}
	responseHeaderValue := strings.TrimSpace(raw.ResponseHeaderValue)
	if responseHeaderValue == "" {
		responseHeaderValue = defaultSubscriptionDecryptResponseHeaderValue
	}
	return &DecryptOptions{
		key:                 key,
		responseHeader:      responseHeader,
		responseHeaderValue: responseHeaderValue,
	}, nil
}

func maybeDecryptSubscriptionContent(header http.Header, content []byte, decrypt *DecryptOptions) ([]byte, error) {
	if decrypt == nil {
		if strings.TrimSpace(header.Get(defaultSubscriptionDecryptResponseHeader)) == defaultSubscriptionDecryptResponseHeaderValue {
			return nil, E.New("subscription response is encrypted, configure subscription.decrypt.key")
		}
		return content, nil
	}
	if !decrypt.ShouldDecrypt(header) {
		return content, nil
	}
	return decrypt.Decrypt(content)
}

func (d *DecryptOptions) ShouldDecrypt(header http.Header) bool {
	if d == nil {
		return false
	}
	return strings.TrimSpace(header.Get(d.responseHeader)) == d.responseHeaderValue
}

func (d *DecryptOptions) Decrypt(content []byte) ([]byte, error) {
	blobBase64 := bytes.Join(bytes.Fields(content), nil)
	blob := make([]byte, base64.StdEncoding.DecodedLen(len(blobBase64)))
	n, err := base64.StdEncoding.Decode(blob, blobBase64)
	if err != nil {
		return nil, E.Cause(err, "decode encrypted content")
	}
	blob = blob[:n]
	if len(blob) < aes.BlockSize {
		return nil, E.New("encrypted content is too short")
	}
	iv, ciphertext := blob[:aes.BlockSize], blob[aes.BlockSize:]
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return nil, E.New("ciphertext length must be a positive multiple of 16")
	}
	block, err := aes.NewCipher(d.key)
	if err != nil {
		return nil, err
	}
	plaintextPadded := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintextPadded, ciphertext)
	plaintext, err := pkcs7Unpad(plaintextPadded, aes.BlockSize)
	if err != nil {
		return nil, E.Cause(err, "unpad plaintext")
	}
	return plaintext, nil
}

func pkcs7Unpad(content []byte, blockSize int) ([]byte, error) {
	if len(content) == 0 || len(content)%blockSize != 0 {
		return nil, E.New("invalid padded plaintext length")
	}
	padding := int(content[len(content)-1])
	if padding == 0 || padding > blockSize || padding > len(content) {
		return nil, E.New("invalid PKCS7 padding size")
	}
	for _, b := range content[len(content)-padding:] {
		if int(b) != padding {
			return nil, E.New("invalid PKCS7 padding content")
		}
	}
	return content[:len(content)-padding], nil
}
