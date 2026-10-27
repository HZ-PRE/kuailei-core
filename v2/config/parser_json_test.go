package config

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"errors"
	"strings"
	"testing"

	"github.com/sagernet/sing-box/experimental/libbox"
)

func TestJSONConfigRejectsAmbiguousInput(t *testing.T) {
	for _, input := range []string{
		`{"outbounds":[],"outbounds":[]}`,
		`{"outbounds":[{"type":"direct","type":"direct"}]}`,
		`{"outbounds":[],"discarded":{"a":1,"a":2}}`,
		`{"outbounds":[]} {"outbounds":[]}`,
		`{"outbounds":[]} trailing`,
		`{"outbounds":[],}`,
		"{\"outbounds\":[],\"bad\":\"\xff\"}",
	} {
		t.Run(input, func(t *testing.T) {
			_, err := ParseConfig(libbox.BaseContext(nil), &ReadOptions{Content: input}, false, nil, false)
			if err == nil || !strings.Contains(err.Error(), "invalid JSON") {
				t.Fatalf("expected JSON error, got %v", err)
			}
		})
	}
}

func TestNormalizeJSONPreservesNumbers(t *testing.T) {
	input := []byte(`{"outbounds":[{"value":9007199254740993}],"route":{"rules":[{"value":9223372036854775807}],"final":"discard"}}`)
	result, err := normalizeJSONConfig(jsontext.NewDecoder(bytes.NewReader(input)), false)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"9007199254740993", "9223372036854775807"} {
		if !bytes.Contains(result, []byte(want)) {
			t.Fatalf("integer changed: %s", result)
		}
	}
	if bytes.Contains(result, []byte("final")) {
		t.Fatal("route.final retained")
	}
}

func TestJSONConfigCommentsAndCancellation(t *testing.T) {
	ctx := libbox.BaseContext(nil)
	_, err := ParseConfig(ctx, &ReadOptions{Content: "/* comment */ {\"outbounds\":[{\"type\":\"direct\"}]}"}, false, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	_, err = ParseConfig(canceled, &ReadOptions{Content: `{"outbounds":[]}`}, false, nil, false)
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
