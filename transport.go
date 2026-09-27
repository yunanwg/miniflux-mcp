package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

const (
	transportStdio          = "stdio"
	transportStreamableHTTP = "streamable-http"
	defaultHTTPAddr         = ":8080"
	defaultHTTPPath         = "/mcp"
)

type transportConfig struct {
	Transport string
	HTTPAddr  string
	HTTPPath  string
	AuthToken string
	// Cloudflare Access: when both are set, a request carrying a valid
	// Cf-Access-Jwt-Assertion for this team and application is accepted.
	AccessTeamDomain string
	AccessAUD        string
}

func loadTransportConfig() (transportConfig, error) {
	cfg := transportConfig{
		Transport: envOrDefault("MCP_TRANSPORT", transportStdio),
		HTTPAddr:  envOrDefault("MCP_HTTP_ADDR", defaultHTTPAddr),
		HTTPPath:  envOrDefault("MCP_HTTP_PATH", defaultHTTPPath),
		AuthToken: os.Getenv("MCP_AUTH_TOKEN"),

		AccessTeamDomain: normalizeTeamDomain(os.Getenv("MCP_CF_ACCESS_TEAM_DOMAIN")),
		AccessAUD:        os.Getenv("MCP_CF_ACCESS_AUD"),
	}

	switch cfg.Transport {
	case transportStdio:
		return cfg, nil
	case transportStreamableHTTP:
		if (cfg.AccessTeamDomain == "") != (cfg.AccessAUD == "") {
			return transportConfig{}, fmt.Errorf("MCP_CF_ACCESS_TEAM_DOMAIN and MCP_CF_ACCESS_AUD must be set together")
		}
		if cfg.AuthToken == "" && cfg.AccessAUD == "" {
			return transportConfig{}, fmt.Errorf("MCP_AUTH_TOKEN or MCP_CF_ACCESS_TEAM_DOMAIN/MCP_CF_ACCESS_AUD is required when MCP_TRANSPORT=%s", transportStreamableHTTP)
		}
		if !strings.HasPrefix(cfg.HTTPPath, "/") || cfg.HTTPPath == "/" {
			return transportConfig{}, fmt.Errorf("MCP_HTTP_PATH must start with / and cannot be /")
		}
		if cfg.HTTPPath == "/healthz" {
			return transportConfig{}, fmt.Errorf("MCP_HTTP_PATH cannot be /healthz")
		}
		return cfg, nil
	default:
		return transportConfig{}, fmt.Errorf("unsupported MCP_TRANSPORT %q (supported: %s, %s)", cfg.Transport, transportStdio, transportStreamableHTTP)
	}
}

// normalizeTeamDomain accepts "team", "team.cloudflareaccess.com" or
// "https://team.cloudflareaccess.com/" and returns the token issuer form,
// "https://team.cloudflareaccess.com".
func normalizeTeamDomain(value string) string {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" {
		return ""
	}
	if !strings.Contains(value, "://") {
		if !strings.Contains(value, ".") {
			value += ".cloudflareaccess.com"
		}
		value = "https://" + value
	}
	return value
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func serveMCP(mcpServer *server.MCPServer, cfg transportConfig) error {
	switch cfg.Transport {
	case transportStdio:
		return server.ServeStdio(mcpServer)
	case transportStreamableHTTP:
		return serveStreamableHTTP(mcpServer, cfg)
	default:
		return fmt.Errorf("unsupported MCP transport %q", cfg.Transport)
	}
}

func serveStreamableHTTP(mcpServer *server.MCPServer, cfg transportConfig) error {
	mcpHandler := server.NewStreamableHTTPServer(
		mcpServer,
		server.WithStateLess(true),
	)

	mux := http.NewServeMux()
	var authenticators []authenticator
	if cfg.AuthToken != "" {
		authenticators = append(authenticators, bearerTokenAuthenticator(cfg.AuthToken))
	}
	if cfg.AccessAUD != "" {
		authenticators = append(authenticators, cloudflareAccessAuthenticator(context.Background(), cfg.AccessTeamDomain, cfg.AccessAUD))
	}

	mux.Handle(cfg.HTTPPath, requireAuth(authenticators, mcpHandler))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	return httpServer.ListenAndServe()
}

// An authenticator reports whether a request carries a valid credential.
type authenticator func(r *http.Request) bool

// requireAuth admits a request if any authenticator accepts it.
func requireAuth(authenticators []authenticator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, authenticate := range authenticators {
			if authenticate(r) {
				next.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	})
}

func bearerTokenAuthenticator(token string) authenticator {
	expectedTokenHash := sha256.Sum256([]byte(token))

	return func(r *http.Request) bool {
		scheme, providedToken, ok := strings.Cut(r.Header.Get("Authorization"), " ")
		providedTokenHash := sha256.Sum256([]byte(providedToken))
		return ok && strings.EqualFold(scheme, "Bearer") && subtle.ConstantTimeCompare(providedTokenHash[:], expectedTokenHash[:]) == 1
	}
}
