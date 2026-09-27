package main

import (
	"context"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
)

// cloudflareAccessHeader carries the application token Cloudflare Access adds
// to every request it lets through to the origin.
const cloudflareAccessHeader = "Cf-Access-Jwt-Assertion"

// cloudflareAccessAuthenticator accepts requests whose Cloudflare Access token
// is signed by the team's current keys, issued by the team domain, unexpired,
// and scoped to the application with this AUD tag.
//
// Verifying the token here, instead of trusting that the request came through
// Cloudflare, keeps the endpoint closed to anything else that can reach the
// origin directly, such as other containers on the same network. It is also
// what makes a single door work: Access does not forward the client's
// Authorization header, so an OAuth-authenticated client cannot also present
// MCP_AUTH_TOKEN.
func cloudflareAccessAuthenticator(ctx context.Context, teamDomain, aud string) authenticator {
	keySet := oidc.NewRemoteKeySet(ctx, teamDomain+"/cdn-cgi/access/certs")
	return cloudflareAccessVerifier(oidc.NewVerifier(teamDomain, keySet, &oidc.Config{ClientID: aud}))
}

func cloudflareAccessVerifier(verifier *oidc.IDTokenVerifier) authenticator {
	return func(r *http.Request) bool {
		token := r.Header.Get(cloudflareAccessHeader)
		if token == "" {
			return false
		}
		_, err := verifier.Verify(r.Context(), token)
		return err == nil
	}
}
