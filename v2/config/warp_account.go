package config

type WarpAccount struct {
	AccountID   string `json:"account-id,omitempty"`
	AccessToken string `json:"access-token,omitempty"`
}

type WarpWireguardConfig struct {
	PrivateKey       string `json:"private-key,omitempty"`
	LocalAddressIPv4 string `json:"local-address-ipv4,omitempty"`
	LocalAddressIPv6 string `json:"local-address-ipv6,omitempty"`
	PeerPublicKey    string `json:"peer-public-key,omitempty"`
	ClientID         string `json:"client-id,omitempty"`
}
