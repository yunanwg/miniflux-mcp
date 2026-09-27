package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	jose "github.com/go-jose/go-jose/v4"
)

const (
	testTeamDomain = "https://example.cloudflareaccess.com"
	testAUD        = "test-aud"
)

func signTestToken(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, nil)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := signer.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}
	token, err := signed.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestCloudflareAccessVerifier(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	keySet := &oidc.StaticKeySet{PublicKeys: []crypto.PublicKey{key.Public()}}
	authenticate := cloudflareAccessVerifier(oidc.NewVerifier(testTeamDomain, keySet, &oidc.Config{ClientID: testAUD}))

	now := time.Now()
	valid := map[string]any{
		"iss": testTeamDomain,
		"aud": []string{testAUD},
		"sub": "user",
		"iat": now.Unix(),
		"exp": now.Add(time.Hour).Unix(),
	}
	with := func(field string, value any) map[string]any {
		claims := make(map[string]any, len(valid))
		for k, v := range valid {
			claims[k] = v
		}
		claims[field] = value
		return claims
	}

	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{"valid token", signTestToken(t, key, valid), true},
		{"no token", "", false},
		{"other application", signTestToken(t, key, with("aud", []string{"other-aud"})), false},
		{"other team", signTestToken(t, key, with("iss", "https://other.cloudflareaccess.com")), false},
		{"expired", signTestToken(t, key, with("exp", now.Add(-time.Minute).Unix())), false},
		{"signed by another key", signTestToken(t, otherKey, valid), false},
		{"garbage", "not-a-jwt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/mcp", nil)
			if tt.token != "" {
				request.Header.Set(cloudflareAccessHeader, tt.token)
			}
			if got := authenticate(request); got != tt.want {
				t.Errorf("authenticate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeTeamDomain(t *testing.T) {
	for input, want := range map[string]string{
		"":                                      "",
		"example":                               testTeamDomain,
		"example.cloudflareaccess.com":          testTeamDomain,
		"https://example.cloudflareaccess.com/": testTeamDomain,
	} {
		if got := normalizeTeamDomain(input); got != want {
			t.Errorf("normalizeTeamDomain(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestLoadTransportConfigAuth(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
	}{
		{"no auth", map[string]string{}, true},
		{"bearer only", map[string]string{"MCP_AUTH_TOKEN": "secret"}, false},
		{"access only", map[string]string{"MCP_CF_ACCESS_TEAM_DOMAIN": "example", "MCP_CF_ACCESS_AUD": testAUD}, false},
		{"access without aud", map[string]string{"MCP_CF_ACCESS_TEAM_DOMAIN": "example"}, true},
		{"both", map[string]string{"MCP_AUTH_TOKEN": "secret", "MCP_CF_ACCESS_TEAM_DOMAIN": "example", "MCP_CF_ACCESS_AUD": testAUD}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MCP_TRANSPORT", transportStreamableHTTP)
			for _, name := range []string{"MCP_AUTH_TOKEN", "MCP_CF_ACCESS_TEAM_DOMAIN", "MCP_CF_ACCESS_AUD"} {
				t.Setenv(name, tt.env[name])
			}
			_, err := loadTransportConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("loadTransportConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
