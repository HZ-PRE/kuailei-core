// Package configvault protects persisted proxy configuration. Traffic uses the
// already parsed in-memory options; encryption runs only on save/load.
package configvault

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var magic = []byte("SDMCFG\x00\x01")
var state struct {
	sync.RWMutex
	aead      cipher.AEAD
	directory string
}

// Initialize stores the random per-installation key in the private base
// directory, never Android's external working directory. Windows additionally
// protects the key with the current user's DPAPI credentials.
func Initialize(base string) error {
	dir := filepath.Join(base, "config-vault")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	unlock, err := lockKey(filepath.Join(dir, "key.lock"))
	if err != nil {
		return err
	}
	defer unlock()
	path := filepath.Join(dir, "key")
	stored, err := os.ReadFile(path)
	var key []byte
	if os.IsNotExist(err) {
		key = make([]byte, 32)
		if _, err = rand.Read(key); err != nil {
			return err
		}
		stored, err = protectKey(key)
		if err == nil {
			err = atomicWrite(path, stored)
		}
	} else if err == nil {
		key, err = unprotectKey(stored)
	}
	if err != nil {
		return fmt.Errorf("configuration key unavailable: %w", err)
	}
	defer clear(key)
	if len(key) != 32 {
		return errors.New("invalid configuration key")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	state.Lock()
	state.aead = aead
	state.directory = dir
	state.Unlock()
	return nil
}

func IsEncrypted(data []byte) bool { return bytes.HasPrefix(data, magic) }

func cryptor() (cipher.AEAD, error) {
	state.RLock()
	defer state.RUnlock()
	if state.aead == nil {
		return nil, errors.New("configuration vault is not initialized")
	}
	return state.aead, nil
}

func Seal(data []byte, purpose string) ([]byte, error) {
	aead, err := cryptor()
	if err != nil {
		return nil, err
	}
	result := make([]byte, len(magic)+aead.NonceSize(), len(magic)+aead.NonceSize()+len(data)+aead.Overhead())
	copy(result, magic)
	nonce := result[len(magic):]
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return aead.Seal(result, nonce, data, append([]byte(purpose), magic...)), nil
}

func Open(data []byte, purpose string) ([]byte, error) {
	aead, err := cryptor()
	if err != nil {
		return nil, err
	}
	start := len(magic)
	if !IsEncrypted(data) || len(data) < start+aead.NonceSize()+aead.Overhead() {
		return nil, errors.New("invalid encrypted configuration")
	}
	plain, err := aead.Open(nil, data[start:start+aead.NonceSize()], data[start+aead.NonceSize():], append([]byte(purpose), magic...))
	if err != nil {
		return nil, errors.New("configuration authentication failed")
	}
	return plain, nil
}

func ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	parent := filepath.Base(filepath.Dir(path))
	if IsEncrypted(data) || filepath.Ext(path) == ".bin" || parent == "configs" || parent == "profiles" {
		return Open(data, "profile")
	}
	// Legacy/import input may be read; application-owned files are migrated at
	// bootstrap and every write below is authenticated ciphertext.
	return data, nil
}

func WriteFile(path string, plain []byte) error {
	data, err := Seal(plain, "profile")
	if err != nil {
		return err
	}
	return atomicWrite(path, data)
}

func atomicWrite(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".sdm-encrypted-*")
	if err != nil {
		return err
	}
	temp := file.Name()
	defer os.Remove(temp)
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(temp, path)
}
