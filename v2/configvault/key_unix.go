//go:build !windows

package configvault

import (
	"golang.org/x/sys/unix"
	"os"
)

func protectKey(key []byte) ([]byte, error)   { return key, nil }
func unprotectKey(key []byte) ([]byte, error) { return key, nil }
func lockKey(path string) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return func() { unix.Flock(int(f.Fd()), unix.LOCK_UN); f.Close() }, nil
}
