package configvault

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MigrateManagedFiles only touches app-owned configuration directories. Old
// names stay valid for Android/iOS background-service restoration.
func migrationMarker(kind, path string) (string, error) {
	state.RLock()
	directory := state.directory
	state.RUnlock()
	if directory == "" {
		return "", fmt.Errorf("configuration vault is not initialized")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, fmt.Sprintf("migration-%x", sha256.Sum256([]byte(kind+":"+absPath)))), nil
}

func MigrationComplete(kind, path string) (bool, error) {
	marker, err := migrationMarker(kind, path)
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(marker); err == nil {
		return true, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	return false, nil
}

func MarkMigrationComplete(kind, path string) error {
	marker, err := migrationMarker(kind, path)
	if err != nil {
		return err
	}
	return atomicWrite(marker, []byte{1})
}

func MigrateManagedFiles(working string) error {
	complete, err := MigrationComplete("profiles", working)
	if err != nil || complete {
		return err
	}
	serverName := regexp.MustCompile(`^server-[a-f0-9]{64}\.json$`)
	for _, dir := range []string{"configs", "data/profiles"} {
		root := filepath.Join(working, dir)
		entries, err := os.ReadDir(root)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.Type()&fs.ModeSymlink != 0 || !entry.Type().IsRegular() {
				continue
			}
			ext := filepath.Ext(entry.Name())
			if ext != ".json" && ext != ".info" {
				continue
			}
			path := filepath.Join(root, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if !IsEncrypted(data) {
				if err := WriteFile(path, data); err != nil {
					return err
				}
				data, err = os.ReadFile(path)
				if err != nil {
					return err
				}
			}
			// Keep the old encrypted path for native system reconnect and seed
			// the new name so the Dart etag cache remains usable after upgrade.
			if dir == "configs" && serverName.MatchString(entry.Name()) {
				target := strings.TrimSuffix(path, ".json") + ".bin"
				if _, err := os.Stat(target); os.IsNotExist(err) {
					if err := atomicWrite(target, data); err != nil {
						return err
					}
				} else if err != nil {
					return err
				}
			}
		}
	}
	// This diagnostic copy is redundant; the running core already owns options.
	err = os.Remove(filepath.Join(working, "data/current-config.json"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return MarkMigrationComplete("profiles", working)
}
