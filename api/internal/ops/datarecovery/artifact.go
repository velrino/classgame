package datarecovery

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

type encryptedArtifact struct {
	Version    int    `json:"version"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

func outsideRepository(path string) bool {
	dir, e := filepath.Abs(filepath.Dir(path))
	if e != nil {
		return false
	}
	dir, e = filepath.EvalSymlinks(dir)
	if e != nil {
		return false
	}
	for {
		if _, e := os.Stat(filepath.Join(dir, ".git")); e == nil {
			return false
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return true
		}
		dir = parent
	}
}
func key(path string) ([]byte, error) {
	real, e := filepath.EvalSymlinks(path)
	if e != nil || !outsideRepository(real) {
		return nil, errors.New("backup key must be outside Git")
	}
	info, e := os.Stat(path)
	if e != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() != 32 {
		return nil, errors.New("backup key must be an owner-only 32-byte file")
	}
	return os.ReadFile(path)
}
func gcm(secret []byte) (cipher.AEAD, error) {
	block, e := aes.NewCipher(secret)
	if e != nil {
		return nil, e
	}
	return cipher.NewGCM(block)
}
func WriteEncrypted(path, keyPath string, b Bundle) error {
	if e := b.Validate(); e != nil {
		return e
	}
	if !outsideRepository(path) || !outsideRepository(keyPath) {
		return errors.New("private backups and keys must be outside a Git repository")
	}
	secret, e := key(keyPath)
	if e != nil {
		return e
	}
	cipher, e := gcm(secret)
	if e != nil {
		return e
	}
	nonce := make([]byte, cipher.NonceSize())
	if _, e = rand.Read(nonce); e != nil {
		return e
	}
	raw, e := json.Marshal(b)
	if e != nil {
		return e
	}
	artifact := encryptedArtifact{Version: 1, Nonce: nonce, Ciphertext: cipher.Seal(nil, nonce, raw, []byte("classgame-class-backup-v1"))}
	file, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	if e = json.NewEncoder(file).Encode(artifact); e != nil {
		file.Close()
		return e
	}
	return file.Close()
}
func ReadEncrypted(path, keyPath string) (Bundle, error) {
	var b Bundle
	info, e := os.Stat(path)
	if e != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 12*1024*1024 {
		return b, errors.New("backup must be an owner-only bounded regular file")
	}
	file, e := os.Open(path)
	if e != nil {
		return b, e
	}
	defer file.Close()
	var artifact encryptedArtifact
	decoder := json.NewDecoder(io.LimitReader(file, 12*1024*1024+1))
	decoder.DisallowUnknownFields()
	if e = decoder.Decode(&artifact); e != nil {
		return b, e
	}
	if decoder.Decode(new(any)) != io.EOF {
		return b, errors.New("backup requires one JSON envelope")
	}
	if artifact.Version != 1 {
		return b, errors.New("unknown backup version")
	}
	secret, e := key(keyPath)
	if e != nil {
		return b, e
	}
	cipher, e := gcm(secret)
	if e != nil {
		return b, e
	}
	if len(artifact.Nonce) != cipher.NonceSize() {
		return b, errors.New("invalid backup nonce")
	}
	raw, e := cipher.Open(nil, artifact.Nonce, artifact.Ciphertext, []byte("classgame-class-backup-v1"))
	if e != nil {
		return b, errors.New("backup authentication failed")
	}
	if len(raw) > MaxBundleBytes {
		return b, errors.New("backup is too large")
	}
	if e = json.Unmarshal(raw, &b); e != nil {
		return b, e
	}
	return b, b.Validate()
}
