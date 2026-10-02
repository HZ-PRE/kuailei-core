package db

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/HZ-PRE/kuailei-core/v2/configvault"
	"github.com/syndtr/goleveldb/leveldb/opt"
	tmdb "github.com/tendermint/tm-db"
)

var directoryState = struct {
	sync.RWMutex
	path string
}{path: "./data"}

func SetDirectory(path string) {
	directoryState.Lock()
	directoryState.path = path
	directoryState.Unlock()
}

func dataDirectory() string {
	directoryState.RLock()
	defer directoryState.RUnlock()
	return directoryState.path
}

// getDB initializes the database with retry logic. If it fails after 100 attempts, it returns nil.
func getDB(name string, readOnly bool) (tmdb.DB, error) {
	// Check if the database file exists; if not, set to readOnly
	directory := dataDirectory()
	dbPath := filepath.Join(directory, name+".db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		readOnly = false
	}
	const retryAttempts = 100
	const retryDelay = 50 * time.Millisecond

	var db tmdb.DB
	var err error
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic: %v", r)
		}
	}()
	for i := 0; i < retryAttempts; i++ {
		// Set readOnly to true for the first 80 attempts
		opts := &opt.Options{ReadOnly: readOnly && i < 80}

		db, err = tmdb.NewGoLevelDBWithOpts(name, directory, opts)
		if err == nil {
			return db, nil
		}
		log.Printf("Failed attempt %d to initialize the database: %v", i, err)
		time.Sleep(retryDelay)
	}
	return nil, err
}

// GetTable returns a new Table instance for the generic type T, ensuring the struct has an "Id" field.
func GetTable[T any]() *Table[T] {
	var t T
	typeName := reflect.TypeOf(t).Name()
	if !hasIdField(t) {
		panic(fmt.Sprintf("Table %s must have a field named 'Id'", typeName))
	}
	return &Table[T]{name: typeName}
}

// hasIdField checks if the struct has a field named "Id".
func hasIdField[T any](t T) bool {
	val := reflect.Indirect(reflect.ValueOf(t))

	if val.Kind() != reflect.Struct {
		return false
	}

	return val.FieldByName("Id").IsValid()
}

// getIdBytes converts an ID to its byte representation for storage in the database.
func getIdBytes(id any) []byte {
	res, err := SerializeKey(id)
	if err != nil {
		return nil
	}
	return res
}

// 	if id == nil {
// 		return nil
// 	}

// 	var buf bytes.Buffer
// 	switch v := id.(type) {
// 	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
// 		if err := binary.Write(&buf, binary.BigEndian, v); err == nil {
// 			return buf.Bytes()
// 		}
// 	case string:
// 		return []byte(v)
// 	case []byte:
// 		return v
// 	default:
// 		return []byte(fmt.Sprint(v))
// 	}
// 	return nil
// }

// getId retrieves the "Id" field from the struct T.
func getId[T any](t T) any {
	val := reflect.Indirect(reflect.ValueOf(t))

	if val.Kind() != reflect.Struct {
		return nil
	}

	field := val.FieldByName("Id")
	if field.IsValid() {
		return field.Interface()
	}
	return nil
}

// Table represents a database table for generic type T.
type Table[T any] struct {
	name string
}

// All retrieves all entries from the database and unmarshals them into a slice of T.
func (tbl *Table[T]) All() ([]*T, error) {
	db, err := getDB(tbl.name, true)
	if db == nil {
		return nil, fmt.Errorf("failed to open database %s, error: %w", tbl.name, err)
	}
	defer db.Close()

	var items []*T
	iter, err := db.Iterator(nil, nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {

		item, err := Deserialize[T](iter.Value())
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func Serialize(data any) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return configvault.Seal(buf.Bytes(), "database")
}

func SerializeKey(data any) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	return buf.Bytes(), err
}

func Deserialize[T any](data []byte) (*T, error) {
	data, err := configvault.Open(data, "database")
	if err != nil {
		return nil, err
	}
	var obj T
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	err = dec.Decode(&obj)
	return &obj, err

	// return &obj, json.Unmarshal(data, &obj)
}

// EncryptExisting upgrades legacy values without changing Gob model names or
// keys. Compaction removes superseded plaintext from live LevelDB files. A
// marker ensures an interrupted migration is compacted on the next bootstrap.
func EncryptExisting() error {
	complete, err := configvault.MigrationComplete("database", dataDirectory())
	if err != nil || complete {
		return err
	}
	entries, err := os.ReadDir(dataDirectory())
	if os.IsNotExist(err) {
		return configvault.MarkMigrationComplete("database", dataDirectory())
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), ".db") {
			continue
		}
		if err := encryptTable(strings.TrimSuffix(entry.Name(), ".db")); err != nil {
			return err
		}
	}
	return configvault.MarkMigrationComplete("database", dataDirectory())
}

func encryptTable(name string) error {
	store, err := getDB(name, false)
	if err != nil {
		return err
	}
	defer store.Close()
	iterator, err := store.Iterator(nil, nil)
	if err != nil {
		return err
	}
	batch := store.NewBatch()
	defer batch.Close()
	changed := false
	for ; iterator.Valid(); iterator.Next() {
		value := iterator.Value()
		if configvault.IsEncrypted(value) {
			continue
		}
		sealed, sealErr := configvault.Seal(value, "database")
		if sealErr != nil {
			iterator.Close()
			return sealErr
		}
		if err = batch.Set(iterator.Key(), sealed); err != nil {
			iterator.Close()
			return err
		}
		changed = true
	}
	err = iterator.Error()
	iterator.Close()
	if err != nil {
		return err
	}
	marker := filepath.Join(dataDirectory(), name+".vault-migration")
	if changed {
		if err := os.WriteFile(marker, []byte{1}, 0600); err != nil {
			return err
		}
		if err := batch.WriteSync(); err != nil {
			return err
		}
	}
	if _, err := os.Stat(marker); err == nil {
		if err := store.(*tmdb.GoLevelDB).ForceCompact(nil, nil); err != nil {
			return err
		}
		return os.Remove(marker)
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

// UpdateInsert inserts or updates multiple items in the database.
func (tbl *Table[T]) UpdateInsert(items ...*T) error {
	db, err := getDB(tbl.name, false)
	if db == nil {
		return fmt.Errorf("failed to open database %s, error: %w", tbl.name, err)
	}
	defer db.Close()

	for _, item := range items {
		// b, err := json.Marshal(item)
		b, err := Serialize(item)
		if err != nil {
			return err
		}
		if err := db.Set(getIdBytes(getId(item)), b); err != nil {
			return err
		}
	}
	return nil
}

// Delete removes entries by their IDs.
func (tbl *Table[T]) Delete(ids ...any) error {
	db, err := getDB(tbl.name, false)
	if db == nil {
		return fmt.Errorf("failed to open database %s, error: %w", tbl.name, err)
	}
	defer db.Close()

	for _, id := range ids {
		if err := db.Delete(getIdBytes(id)); err != nil {
			return err
		}
	}
	return nil
}

// Get retrieves a single item by its ID.
func (tbl *Table[T]) Get(id any) (*T, error) {
	db, err := getDB(tbl.name, true)
	if db == nil {
		return nil, fmt.Errorf("failed to open database %s, error: %w", tbl.name, err)
	}
	defer db.Close()

	b, err := db.Get(getIdBytes(id))
	if err != nil {
		return nil, err
	}
	return Deserialize[T](b)
}
