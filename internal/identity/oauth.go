package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// oauthProvider abstracts an OAuth 2.0 / OIDC provider (ADR-0006). v1 ships GitHub
// (plain OAuth; user id via api.github.com); the abstraction stays generic so an
// OIDC provider like Google slots in later. The Authorization Code flow uses state
// (CSRF) + PKCE (S256); GitHub ignores PKCE but requires the client secret on
// exchange, so both are sent.
type oauthProvider struct {
	name        string
	authURL     string
	tokenURL    string
	userInfoURL string
	scopes      []string
	client      OAuthClient
}

// profile is the normalized identity read from a provider.
type profile struct {
	ProviderUserID string
	DisplayName    string
	Email          string
}

func newProviders(cfg AuthConfig) map[string]*oauthProvider {
	// v1 ships GitHub only; Google (ADR-0006) is deferred until its OAuth app is
	// registered — re-add it here plus a googleProfile branch in userProfile.
	return map[string]*oauthProvider{
		"github": {
			name:        "github",
			authURL:     "https://github.com/login/oauth/authorize",
			tokenURL:    "https://github.com/login/oauth/access_token",
			userInfoURL: "https://api.github.com/user",
			scopes:      []string{"read:user", "user:email"},
			client:      cfg.GitHub,
		},
	}
}

// authCodeURL builds the provider's authorize URL for the Authorization Code flow.
func (p *oauthProvider) authCodeURL(redirectURI, state, codeChallenge string) string {
	q := url.Values{}
	q.Set("client_id", p.client.ClientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(p.scopes, " "))
	q.Set("state", state)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	return p.authURL + "?" + q.Encode()
}

// exchange swaps an authorization code for an access token (confidential client +
// PKCE verifier).
func (p *oauthProvider) exchange(ctx context.Context, httpc *http.Client, code, redirectURI, codeVerifier string) (string, error) {
	form := url.Values{}
	form.Set("client_id", p.client.ClientID)
	form.Set("client_secret", p.client.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("grant_type", "authorization_code")
	form.Set("code_verifier", codeVerifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json") // GitHub returns form-encoded without this

	resp, err := httpc.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint status %d", resp.StatusCode)
	}
	var tr struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if tr.Error != "" {
		return "", fmt.Errorf("token error: %s", tr.Error)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("token response missing access_token")
	}
	return tr.AccessToken, nil
}

// userProfile fetches and normalizes the provider's user profile.
func (p *oauthProvider) userProfile(ctx context.Context, httpc *http.Client, accessToken string) (profile, error) {
	body, err := p.get(ctx, httpc, p.userInfoURL, accessToken)
	if err != nil {
		return profile{}, err
	}
	switch p.name {
	case "github":
		return p.githubProfile(ctx, httpc, accessToken, body)
	default:
		return profile{}, fmt.Errorf("unknown provider %q", p.name)
	}
}

func (p *oauthProvider) githubProfile(ctx context.Context, httpc *http.Client, accessToken string, body []byte) (profile, error) {
	var u struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return profile{}, fmt.Errorf("decode github user: %w", err)
	}
	if u.ID == 0 {
		return profile{}, fmt.Errorf("github user missing id")
	}
	name := u.Name
	if name == "" {
		name = u.Login
	}
	email := u.Email
	if email == "" {
		email = githubPrimaryEmail(ctx, httpc, accessToken)
	}
	return profile{
		ProviderUserID: strconv.FormatInt(u.ID, 10),
		DisplayName:    name,
		Email:          email,
	}, nil
}

// githubPrimaryEmail best-effort resolves a primary verified email when the public
// profile hides it. Failures are non-fatal (email is optional).
func githubPrimaryEmail(ctx context.Context, httpc *http.Client, accessToken string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := httpc.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&emails); err != nil {
		return ""
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email
		}
	}
	for _, e := range emails {
		if e.Verified {
			return e.Email
		}
	}
	return ""
}

func (p *oauthProvider) get(ctx context.Context, httpc *http.Client, url, accessToken string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	resp, err := httpc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}

// --- PKCE + state helpers ---

func randToken(nBytes int) string {
	b := make([]byte, nBytes)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// newState returns a CSRF state token.
func newState() string { return randToken(24) }

// newPKCE returns a PKCE (verifier, S256 challenge) pair.
func newPKCE() (verifier, challenge string) {
	verifier = randToken(32)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge
}
