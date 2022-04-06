package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha512"
	"io"
)

func NewAesCipher(key []byte) (cipher.Block, error) {
	k := sha512.Sum512_256(key)
	b32, err := aes.NewCipher(k[:])
	if err != nil {
		return nil, err
	}

	return b32, err
}

func EncryptBytes(b cipher.Block, bs []byte) ([]byte, error) {
	if b != nil {
		aesGCM, err := cipher.NewGCM(b)
		if err != nil {
			return nil, err
		}
		nonce := make([]byte, aesGCM.NonceSize())
		if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
			return nil, err
		}

		bs = aesGCM.Seal(nonce, nonce, bs, nil)
	}
	return bs, nil
}

func DecryptBytes(b cipher.Block, bs []byte) ([]byte, error) {
	if b != nil {
		aesGCM, err := cipher.NewGCM(b)
		if err != nil {
			return nil, err
		}
		nonceSize := aesGCM.NonceSize()
		nonce, ciphertext := bs[:nonceSize], bs[nonceSize:]
		if bs, err = aesGCM.Open(nil, nonce, ciphertext, nil); err != nil {
			return nil, err
		}
	}
	return bs, nil
}

type StreamWriter struct {
	cipher.StreamWriter
}

func CipherWriter(b cipher.Block, w io.Writer) io.Writer {
	if b == nil {
		return w
	}
	iv := make([]byte, b.BlockSize())
	stream := cipher.NewOFB(b, iv[:])

	c := cipher.StreamWriter{S: stream, W: w}
	return &StreamWriter{c}
}

type StreamReader struct {
	cipher.StreamReader
}

func CipherReader(b cipher.Block, r io.Reader) io.Reader {
	if b == nil {
		return r
	}

	iv := make([]byte, b.BlockSize())
	stream := cipher.NewOFB(b, iv[:])

	c := cipher.StreamReader{S: stream, R: r}
	return &StreamReader{c}
}
