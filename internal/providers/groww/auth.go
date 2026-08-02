package groww

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

// Authenticator obtains and refreshes Groww access tokens.
type Authenticator struct {
	cfg    Config
	client *Client
	token  string
}

func newAuthenticator(cfg Config, client *Client) *Authenticator {
	return &Authenticator{
		cfg:    cfg,
		client: client,
		token:  cfg.AccessToken,
	}
}

// Token returns the current access token.
func (a *Authenticator) Token() string {
	return a.token
}

// Authenticated reports whether a token is available.
func (a *Authenticator) Authenticated() bool {
	return a.token != ""
}

// Authenticate acquires or validates an access token.
func (a *Authenticator) Authenticate(ctx context.Context) (string, error) {
	if a.token != "" {
		return a.token, nil
	}
	if a.cfg.APIKey == "" || a.cfg.APISecret == "" {
		return "", ErrNotAuthenticated
	}

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	checksum := generateChecksum(a.cfg.APISecret, timestamp)
	body := map[string]any{
		"key_type":  "approval",
		"checksum":  checksum,
		"timestamp": timestamp,
	}

	var resp TokenResponse
	if err := a.client.PostJSON(ctx, "/v1/token/api/access", body, &resp, true); err != nil {
		return "", fmt.Errorf("groww authenticate: %w", err)
	}
	if resp.Token == "" {
		return "", fmt.Errorf("groww authenticate: empty token")
	}
	a.token = resp.Token
	return a.token, nil
}

func generateChecksum(secret, timestamp string) string {
	sum := sha256.Sum256([]byte(secret + timestamp))
	return hex.EncodeToString(sum[:])
}
