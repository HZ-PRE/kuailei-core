package configvault

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPersistenceAndAuthentication(t *testing.T) {
	base := t.TempDir()
	if err := Initialize(base); err != nil {
		t.Fatal(err)
	}
	plain := []byte(`{"outbounds":[{"type":"vless","uuid":"private-fixture-marker"}]}`)
	path := filepath.Join(t.TempDir(), "profile.bin")
	if err := WriteFile(path, plain); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(path)
	if !IsEncrypted(first) || bytes.Contains(first, []byte("private-fixture-marker")) {
		t.Fatal("plaintext persisted")
	}
	if err := WriteFile(path, plain); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(path)
	if bytes.Equal(first, second) {
		t.Fatal("nonce reused")
	}
	if err := Initialize(base); err != nil {
		t.Fatal(err)
	}
	decoded, err := ReadFile(path)
	if err != nil || !bytes.Equal(decoded, plain) {
		t.Fatalf("restart restore failed: %v", err)
	}
	for _, offset := range []int{0, len(magic), len(second) - 1} {
		corrupt := append([]byte(nil), second...)
		corrupt[offset] ^= 1
		if _, err := Open(corrupt, "profile"); err == nil {
			t.Fatal("tamper accepted")
		}
	}
	if _, err := Open(second[:10], "profile"); err == nil {
		t.Fatal("truncation accepted")
	}
	if _, err := Open(second, "database"); err == nil {
		t.Fatal("wrong purpose accepted")
	}
	if err := Initialize(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFile(path); err == nil {
		t.Fatal("another installation decrypted profile")
	}
	if err := Initialize(base); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(filepath.Join(path, "child"), plain); err == nil {
		t.Fatal("invalid output path accepted")
	}
	decoded, err = ReadFile(path)
	if err != nil || !bytes.Equal(decoded, plain) {
		t.Fatal("failed save changed old file")
	}
}

func TestLegacyMigrationAndCleanup(t *testing.T) {
	base, working := t.TempDir(), t.TempDir()
	if err := Initialize(base); err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{"configs/legacy.json", "configs/legacy.tmp.json", "data/profiles/legacy.info", "data/current-config.json"} {
		path := filepath.Join(working, relative)
		os.MkdirAll(filepath.Dir(path), 0700)
		os.WriteFile(path, []byte("private-legacy-fixture"), 0600)
	}
	for i := 0; i < 2; i++ {
		if err := MigrateManagedFiles(working); err != nil {
			t.Fatal(err)
		}
	}
	for _, relative := range []string{"configs/legacy.json", "configs/legacy.tmp.json", "data/profiles/legacy.info"} {
		path := filepath.Join(working, relative)
		stored, _ := os.ReadFile(path)
		if !IsEncrypted(stored) {
			t.Fatal("legacy plaintext retained")
		}
		decoded, err := ReadFile(path)
		if err != nil || string(decoded) != "private-legacy-fixture" {
			t.Fatalf("migration changed content: %v", err)
		}
	}
	if _, err := os.Stat(filepath.Join(working, "data/current-config.json")); !os.IsNotExist(err) {
		t.Fatal("diagnostic config retained")
	}
	entries, _ := os.ReadDir(filepath.Join(working, "configs"))
	if len(entries) != 2 {
		t.Fatal("temporary file leaked")
	}
}

func BenchmarkSeal1MiB(b *testing.B) {
	if err := Initialize(b.TempDir()); err != nil {
		b.Fatal(err)
	}
	data := make([]byte, 1024*1024)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Seal(data, "profile"); err != nil {
			b.Fatal(err)
		}
	}
}

func TestServerCacheUpgradeKeepsNativeAndDartPaths(t *testing.T) {
	working := t.TempDir()
	if err := Initialize(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(working, "configs")
	os.MkdirAll(dir, 0700)
	base := filepath.Join(dir, "server-"+strings.Repeat("a", 64))
	os.WriteFile(base+".json", []byte("upgrade-secret"), 0600)
	if err := MigrateManagedFiles(working); err != nil {
		t.Fatal(err)
	}
	for _, ext := range []string{".json", ".bin"} {
		data, err := ReadFile(base + ext)
		if err != nil || string(data) != "upgrade-secret" {
			t.Fatalf("lost %s: %v", ext, err)
		}
	}
	// A completed migration must never authenticate a newly injected plaintext
	// file, regardless of whether an attacker chooses the old or new extension.
	for _, ext := range []string{".json", ".bin"} {
		os.WriteFile(base+ext, []byte(`{"outbounds":[]}`), 0600)
	}
	if err := MigrateManagedFiles(working); err != nil {
		t.Fatal(err)
	}
	for _, ext := range []string{".json", ".bin"} {
		if _, err := ReadFile(base + ext); err == nil {
			t.Fatalf("plaintext injection accepted: %s", ext)
		}
	}
}
