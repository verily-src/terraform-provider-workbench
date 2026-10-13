package goclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// cliTokenExpiryDelta is how far before a CLI token's actual expiry the in-memory
// cache treats it as stale and re-invokes the CLI. It matches the Workbench CLI's
// own refresh window (expirationWindowSeconds = 300 in cli-go/internal/auth), so
// the provider re-fetches right as `wb` begins refreshing on its side.
const cliTokenExpiryDelta = 300 * time.Second

// sharedCLITokenSource caches the token from the Workbench CLI across every
// client created in this process and refreshes it cliTokenExpiryDelta before
// expiry. A single shared source (rather than one per client) avoids spawning
// the `wb` subprocess on every operation. The underlying reuseTokenSource is
// safe for concurrent use, and oauth2.NewClient returns it unwrapped when
// passed a *reuseTokenSource, so the configured expiry buffer is preserved.
var sharedCLITokenSource = oauth2.ReuseTokenSourceWithExpiry(nil, &cliTokenSource{}, cliTokenExpiryDelta)

// OauthClientFromCLI creates an http client using credentials from the Workbench CLI.
// It calls `wb auth print-access-token` to get the current user's access token.
// The token is cached in memory and automatically refreshed shortly before it expires.
func OauthClientFromCLI(ctx context.Context) (*http.Client, error) {
	return oauth2.NewClient(ctx, sharedCLITokenSource), nil
}

// cliTokenSource implements oauth2.TokenSource by calling the Workbench CLI
type cliTokenSource struct{}

// Token fetches a fresh access token from the CLI
func (c *cliTokenSource) Token() (*oauth2.Token, error) {
	return getTokenFromCLI()
}

// getTokenFromCLI calls `wb auth print-access-token` and parses the JWT
func getTokenFromCLI() (*oauth2.Token, error) {
	cmd := exec.Command("wb", "auth", "print-access-token")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get token from the Workbench CLI (ensure the CLI is in your path and 'wb auth login' has been run): %w", err)
	}

	tokenStr := strings.TrimSpace(string(output))
	if tokenStr == "" {
		return nil, fmt.Errorf("CLI returned empty token")
	}

	// Parse JWT to extract expiration
	expiry, err := parseJWTExpiration(tokenStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT expiration: %w", err)
	}

	return &oauth2.Token{
		AccessToken: tokenStr,
		TokenType:   "Bearer",
		Expiry:      expiry,
	}, nil
}

// parseJWTExpiration extracts the expiration time from a JWT token
func parseJWTExpiration(token string) (time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid JWT format: expected 3 parts, got %d", len(parts))
	}

	// Decode the payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}, fmt.Errorf("failed to unmarshal JWT claims: %w", err)
	}

	if claims.Exp == 0 {
		return time.Time{}, fmt.Errorf("JWT does not contain expiration claim")
	}

	return time.Unix(claims.Exp, 0), nil
}
