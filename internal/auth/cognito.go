package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type jwksKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwksResponse struct {
	Keys []jwksKey `json:"keys"`
}

type CognitoConfig struct {
	UserPoolID   string
	ClientID     string
	ClientSecret string
	Domain       string
	Region       string
}

type TokenResponse struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type IDTokenClaims struct {
	Email    string `json:"email"`
	Username string `json:"cognito:username"`
	Sub      string `json:"sub"`
	jwt.RegisteredClaims
}

func (c *CognitoConfig) LoginURL(redirectURI string) string {
	u := fmt.Sprintf("https://%s.auth.%s.amazoncognito.com/login", c.Domain, c.Region)
	v := url.Values{}
	v.Set("client_id", c.ClientID)
	v.Set("response_type", "code")
	v.Set("scope", "openid email profile")
	v.Set("redirect_uri", redirectURI)
	return u + "?" + v.Encode()
}

func (c *CognitoConfig) LogoutURL(redirectURI string) string {
	u := fmt.Sprintf("https://%s.auth.%s.amazoncognito.com/logout", c.Domain, c.Region)
	v := url.Values{}
	v.Set("client_id", c.ClientID)
	v.Set("logout_uri", redirectURI)
	return u + "?" + v.Encode()
}

func (c *CognitoConfig) ExchangeCode(ctx context.Context, code, redirectURI string) (*TokenResponse, error) {
	tokenURL := fmt.Sprintf("https://%s.auth.%s.amazoncognito.com/oauth2/token", c.Domain, c.Region)

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	body := data.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	auth := base64.StdEncoding.EncodeToString([]byte(c.ClientID + ":" + c.ClientSecret))
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(respBody))
	}

	var tr TokenResponse
	if err := json.Unmarshal(respBody, &tr); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	return &tr, nil
}

func (c *CognitoConfig) VerifyIDToken(ctx context.Context, idToken string) (*IDTokenClaims, error) {
	token, err := jwt.ParseWithClaims(idToken, &IDTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return c.getSigningKey(ctx, token)
	})
	if err != nil {
		return nil, fmt.Errorf("parse id token: %w", err)
	}

	claims, ok := token.Claims.(*IDTokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid id token claims")
	}

	iss := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", c.Region, c.UserPoolID)
	if claims.Issuer != iss {
		return nil, fmt.Errorf("invalid issuer: %s", claims.Issuer)
	}

	if claims.Audience[0] != c.ClientID {
		return nil, fmt.Errorf("invalid audience")
	}

	return claims, nil
}

func (c *CognitoConfig) getSigningKey(ctx context.Context, token *jwt.Token) (interface{}, error) {
	jwksURL := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", c.Region, c.UserPoolID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create jwks request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read jwks: %w", err)
	}

	var jwks jwksResponse
	if err := json.Unmarshal(body, &jwks); err != nil {
		return nil, fmt.Errorf("parse jwks: %w", err)
	}

	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("no kid in token header")
	}

	for _, key := range jwks.Keys {
		if key.Kid == kid {
			return parseRSAPublicKey(key.N, key.E)
		}
	}

	return nil, fmt.Errorf("no matching key for kid: %s", kid)
}

func parseRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}
