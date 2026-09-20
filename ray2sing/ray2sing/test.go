package ray2sing

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/sagernet/sing-box/experimental/libbox"
)

func CheckUrlAndJson(url string, expectedJSON string, t *testing.T) {
	configJson, err := Ray2Singbox(libbox.BaseContext(nil), url, false)
	if err != nil {
		t.Fatalf("Error parsing URL: %v", err)
	}

	// Convert the expected JSON to a comparable Go structure
	expectedConfig, expectedPretty, err := json2map_prettystr(expectedJSON)
	if err != nil {
		t.Fatalf("Failed to unmarshal expected JSON: %v \n%v", err, expectedPretty)
	}
	config, configPretty, err := json2map_prettystr(string(configJson))
	if err != nil {
		t.Fatalf("Failed to unmarshal config JSON: %v \n%v", err, configPretty)
	}

	// Compare the actual options with the expected configuration
	if !reflect.DeepEqual(config, expectedConfig) {
		t.Errorf("Parsed options do not match expected configuration. Got \n%+v, \n\n =====want====\n%+v", configPretty, expectedPretty)
	}
}

func json2map_prettystr(injson string) (map[string]any, string, error) {
	var conf map[string]any
	if err := json.Unmarshal([]byte(injson), &conf); err != nil {
		return nil, "", err
	}
	if conf["outbounds"] == nil && conf["endpoints"] == nil {
		return nil, "", fmt.Errorf("no outbounds or endpoints")
	}
	pp, err := json.MarshalIndent(conf, "", " ")
	return conf, string(pp), err
}
func sortedMarshal(data map[string]interface{}) (string, error) {
	// Create a slice for storing sorted keys
	var keys []string
	for k := range data {
		keys = append(keys, k)
	}

	// Sort the keys
	sort.Strings(keys)

	// Create a new map to hold sorted data
	sortedData := make(map[string]interface{}, len(data))
	for _, k := range keys {
		sortedData[k] = data[k]
	}

	// Marshal the sorted map with indentation
	jsonBytes, err := json.MarshalIndent(sortedData, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}
