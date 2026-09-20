package ray2sing_test

import (
	"github.com/HZ-PRE/ray2sing/ray2sing"
	"testing"
)

// WireGuard links now map to endpoints rather than legacy outbounds.
func TestWireGuardEndpoint(t *testing.T) {
	url := "wg://server.example:222/?pk=[private_key]&local_address=10.0.0.2/24&peer_public_key=[peer_public_key]&pre_shared_key=[pre_shared_key]&workers=2&mtu=1280&reserved=0,0,0"
	expectedJSON := `{
  "endpoints": [{
   "type": "wireguard", "tag": "WG § 0", "address": "10.0.0.2/24",
   "private_key": "[private_key]", "workers": 2, "mtu": 1280,
   "noise": {"fake_packet": {"count": "", "delay": "", "size": ""}},
   "peers": [{"address": "server.example", "port": 222,
    "public_key": "[peer_public_key]", "pre_shared_key": "[pre_shared_key]",
    "allowed_ips": ["0.0.0.0/0", "::/0"], "reserved": "AAAA"}]
  }]
 }`
	ray2sing.CheckUrlAndJson(url, expectedJSON, t)
}
