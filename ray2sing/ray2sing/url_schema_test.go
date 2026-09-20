package ray2sing

import (
	"encoding/base64"
	"testing"
)

func TestParseURLNormalizesParameterNames(t *testing.T) {
	u, err := ParseUrl("tuic://user:pass@example.com?congestion_control=bbr&obfs-password=test&allow_insecure=1", 443)
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"congestioncontrol": "bbr", "obfspassword": "test", "allowinsecure": "1"} {
		if u.Params[key] != want {
			t.Errorf("%s: got %q, want %q", key, u.Params[key], want)
		}
	}
}

func TestParseURLPreservesColonInEncodedPassword(t *testing.T) {
	userinfo := base64.StdEncoding.EncodeToString([]byte("2022-blake3-aes-128-gcm:first:second"))
	u, err := ParseUrl("ss://"+userinfo+"@example.com:443", 443)
	if err != nil {
		t.Fatal(err)
	}
	if u.Username != "2022-blake3-aes-128-gcm" || u.Password != "first:second" {
		t.Fatalf("encoded user info was not preserved: %+v", u)
	}
}
