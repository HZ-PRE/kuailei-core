package db_test

import (
	"encoding/hex"
	"os"
	"testing"

	"github.com/HZ-PRE/kuailei-core/v2/db"
	"github.com/HZ-PRE/kuailei-core/v2/hcommon"
	tmdb "github.com/tendermint/tm-db"
)

// Produced by the normal Go compiler, before identifier obfuscation. Keep this
// fixture literal: generating it with the tested compiler would hide a schema
// rename. Run this test with both go test and the release GOGARBLE selection.
func TestPreObfuscationSettingsRemainReadable(t *testing.T) {
	t.Chdir(t.TempDir())
	const fixture = "297f0301010b41707053657474696e677301ff8000010201024964010c00010556616c7565011000000037ff80010d636f6d7061746962696c6974790106737472696e670c1b001973746f7265642d6265666f72652d6f62667573636174696f6e00"
	value, err := hex.DecodeString(fixture)
	if err != nil {
		t.Fatal(err)
	}
	key, err := db.SerializeKey("compatibility")
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := tmdb.NewGoLevelDB("AppSettings", "./data")
	if err != nil {
		t.Fatal(err)
	}
	if err := legacy.Set(key, value); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	table := db.GetTable[hcommon.AppSettings]()
	record, err := table.Get("compatibility")
	if err != nil {
		t.Fatal(err)
	}
	if record.Id != "compatibility" || record.Value != "stored-before-obfuscation" {
		t.Fatalf("legacy record changed: %#v", record)
	}
	record.Value = "updated"
	if err := table.UpdateInsert(record); err != nil {
		t.Fatal(err)
	}
	reloaded, err := table.Get("compatibility")
	if err != nil || reloaded.Value != "updated" {
		t.Fatalf("record could not be updated: %v", err)
	}
	entries, err := os.ReadDir("data")
	if err != nil || len(entries) != 1 || entries[0].Name() != "AppSettings.db" {
		t.Fatalf("database name changed after obfuscation: %v, %v", entries, err)
	}
}
