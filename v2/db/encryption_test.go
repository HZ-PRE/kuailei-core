package db_test

import (
	"bytes"
	"encoding/gob"
	"os"
	"path/filepath"
	"testing"

	"github.com/HZ-PRE/kuailei-core/v2/configvault"
	"github.com/HZ-PRE/kuailei-core/v2/db"
	"github.com/HZ-PRE/kuailei-core/v2/hcommon"
	tmdb "github.com/tendermint/tm-db"
)

func TestDatabaseMigrationRemovesPlaintextAndPreservesRestore(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := configvault.Initialize(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	const secret = "private-database-proxy-fixture"
	record := hcommon.AppSettings{Id: "lastStartRequestContent", Value: secret}
	var legacy bytes.Buffer
	if err := gob.NewEncoder(&legacy).Encode(record); err != nil {
		t.Fatal(err)
	}
	key, _ := db.SerializeKey(record.Id)
	store, err := tmdb.NewGoLevelDB("AppSettings", "data")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetSync(key, legacy.Bytes()); err != nil {
		t.Fatal(err)
	}
	store.Close()
	for i := 0; i < 2; i++ {
		if err := db.EncryptExisting(); err != nil {
			t.Fatal(err)
		}
	}
	loaded, err := db.GetTable[hcommon.AppSettings]().Get(record.Id)
	if err != nil || loaded.Value != secret {
		t.Fatalf("lost restored config: %v", err)
	}
	if err := db.GetTable[hcommon.AppSettings]().UpdateInsert(&record); err != nil {
		t.Fatal(err)
	}
	err = filepath.WalkDir("data", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(data, []byte(secret)) {
			t.Errorf("plaintext remains in %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	store, err = tmdb.NewGoLevelDB("AppSettings", "data")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetSync(key, legacy.Bytes()); err != nil {
		t.Fatal(err)
	}
	store.Close()
	if err := db.EncryptExisting(); err != nil {
		t.Fatal(err)
	}
	if _, err := db.GetTable[hcommon.AppSettings]().Get(record.Id); err == nil {
		t.Fatal("post-migration plaintext injection accepted")
	}
}
