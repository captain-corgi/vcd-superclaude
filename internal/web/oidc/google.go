package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/captain-corgi/multi-agent-system/pkg/config"
)

// OIDCProvider defines the interface for OIDC providers
type OIDCProvider interface {
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*TokenInfo, error)
	VerifyToken(ctx context.Context, token string) (*UserInfo, error)
	GetProviderName() string
}

// TokenInfo represents the token information from OIDC
type TokenInfo struct {
	AccessToken  string
	IDToken      string
	RefreshToken string
	ExpiresIn    time.Duration
}

// UserInfo represents the user information from OIDC
type UserInfo struct {
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// GoogleOIDCProvider implements OIDCProvider for Google
type GoogleOIDCProvider struct {
	config      *config.GoogleConfig
	oauthConfig *oauth2.Config
	verifier    *oidc.IDTokenVerifier
	httpClient  *http.Client
	logger      *zap.Logger
}

// NewGoogleOIDCProvider creates a new Google OIDC provider
func NewGoogleOIDCProvider(googleConfig *config.GoogleConfig, logger *zap.Logger) (*GoogleOIDCProvider, error) {
	ctx := context.Background()

	// Create OAuth2 config
	oauthConfig := &oauth2.Config{
		ClientID:     googleConfig.ClientID,
		ClientSecret: googleConfig.ClientSecret,
		RedirectURL:  fmt.Sprintf("%s/auth/google/callback", strings.TrimSuffix(googleConfig.RedirectBaseURL, "/")),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		Endpoint:     google.Endpoint,
	}

	// Create OIDC provider
	provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	// Create verifier
	verifier := provider.Verifier(&oidc.Config{ClientID: googleConfig.ClientID})

	googleProvider := &GoogleOIDCProvider{
		config:      googleConfig,
		oauthConfig: oauthConfig,
		verifier:    verifier,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		logger:      logger,
	}

	logger.Info("Google OIDC provider initialized",
		zap.String("client_id", googleConfig.ClientID),
		zap.String("redirect_url", oauthConfig.RedirectURL))

	return googleProvider, nil
}

// GetAuthURL generates the Google OAuth authorization URL
func (p *GoogleOIDCProvider) GetAuthURL(state string) string {
	return p.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// ExchangeCode exchanges the authorization code for tokens
func (p *GoogleOIDCProvider) ExchangeCode(ctx context.Context, code string) (*TokenInfo, error) {
	token, err := p.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	tokenInfo := &TokenInfo{
		AccessToken:  token.AccessToken,
		IDToken:      token.Extra("id_token").(string),
		RefreshToken: token.RefreshToken,
		ExpiresIn:    time.Until(token.Expiry),
	}

	p.logger.Debug("Token exchanged successfully",
		zap.Bool("has_access_token", tokenInfo.AccessToken != ""),
		zap.Bool("has_id_token", tokenInfo.IDToken != ""),
		zap.Duration("expires_in", tokenInfo.ExpiresIn))

	return tokenInfo, nil
}

// VerifyToken verifies and parses the ID token
func (p *GoogleOIDCProvider) VerifyToken(ctx context.Context, token string) (*UserInfo, error) {
	idToken, err := p.verifier.Verify(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	var claims struct {
		Subject       string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		Picture       string `json:"picture"`
		Locale        string `json:"locale"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	userInfo := &UserInfo{
		Subject:       claims.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		Name:          claims.Name,
		GivenName:     claims.GivenName,
		FamilyName:    claims.FamilyName,
		Picture:       claims.Picture,
		Locale:        claims.Locale,
	}

	p.logger.Debug("ID token verified successfully",
		zap.String("subject", userInfo.Subject),
		zap.String("email", userInfo.Email),
		zap.Bool("email_verified", userInfo.EmailVerified))

	return userInfo, nil
}

// GetProviderName returns the provider name
func (p *GoogleOIDCProvider) GetProviderName() string {
	return "google"
}

// ValidateDomain checks if the user's email domain is allowed
func (p *GoogleOIDCProvider) ValidateDomain(email string) error {
	if len(p.config.AllowedDomains) == 0 {
		// No domain restrictions
		return nil
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid email format")
	}

	domain := parts[1]
	for _, allowedDomain := range p.config.AllowedDomains {
		if domain == allowedDomain {
			return nil
		}
	}

	return fmt.Errorf("domain %s not allowed", domain)
}

// GetUserInfoFromGoogle retrieves user info directly from Google API
func (p *GoogleOIDCProvider) GetUserInfoFromGoogle(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: status %d, body: %s", resp.StatusCode, string(body))
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Validate domain if configured
	if err := p.ValidateDomain(userInfo.Email); err != nil {
		return nil, err
	}

	p.logger.Debug("User info retrieved from Google API",
		zap.String("email", userInfo.Email),
		zap.String("name", userInfo.Name))

	return &userInfo, nil
}
