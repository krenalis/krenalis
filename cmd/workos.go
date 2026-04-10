// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"
)

// workosClientIDString returns the WorkOS client ID, or an empty string if
// WorkOS authentication is not configured.
func (s *apisServer) workosClientIDString() string {
	if s.workos == nil {
		return ""
	}
	return s.workos.clientID
}

// workosAuth handles WorkOS JWT verification and user lookup.
type workosAuth struct {
	clientID string
	apiKey   string
}

type workosJWKS struct {
	Keys []struct {
		Kty string `json:"kty"`
		N   string `json:"n"`
		E   string `json:"e"`
		Kid string `json:"kid"`
	} `json:"keys"`
}

// verifyToken verifies the WorkOS access token JWT and returns the authenticated
// user's email. It verifies the RS256 signature using the WorkOS JWKS and then
// calls the WorkOS API to retrieve the user's email by the sub claim.
func (wa *workosAuth) verifyToken(tokenString string) (string, error) {
	// Split the JWT into header, payload and signature.
	parts := strings.SplitN(tokenString, ".", 3)
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid JWT format")
	}
	headerB64, payloadB64, sigB64 := parts[0], parts[1], parts[2]

	// Decode and parse the header to get the algorithm and key ID.
	headerJSON, err := base64.RawURLEncoding.DecodeString(headerB64)
	if err != nil {
		return "", fmt.Errorf("invalid JWT header encoding: %w", err)
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return "", fmt.Errorf("invalid JWT header: %w", err)
	}
	if header.Alg != "RS256" {
		return "", fmt.Errorf("unexpected JWT algorithm %q, expected RS256", header.Alg)
	}

	// Fetch the RSA public key matching the key ID.
	pubKey, err := wa.publicKey(header.Kid)
	if err != nil {
		return "", err
	}

	// Verify the RS256 signature over "headerB64.payloadB64".
	sigBytes, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return "", fmt.Errorf("invalid JWT signature encoding: %w", err)
	}
	digest := sha256.Sum256([]byte(headerB64 + "." + payloadB64))
	if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, digest[:], sigBytes); err != nil {
		return "", fmt.Errorf("invalid JWT signature: %w", err)
	}

	// Decode and parse the payload.
	payloadJSON, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return "", fmt.Errorf("invalid JWT payload encoding: %w", err)
	}
	var claims struct {
		Sub string `json:"sub"`
		Exp int64  `json:"exp"`
		Nbf int64  `json:"nbf"`
	}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return "", fmt.Errorf("invalid JWT payload: %w", err)
	}

	// Validate time-based claims.
	now := time.Now().Unix()
	if claims.Exp > 0 && now > claims.Exp {
		return "", fmt.Errorf("JWT has expired")
	}
	if claims.Nbf > 0 && now < claims.Nbf {
		return "", fmt.Errorf("JWT is not yet valid")
	}
	if claims.Sub == "" {
		return "", fmt.Errorf("missing sub claim in JWT")
	}

	return wa.userEmail(claims.Sub)
}

// publicKey fetches the WorkOS JWKS and returns the RSA public key for the
// given key ID.
func (wa *workosAuth) publicKey(kid string) (*rsa.PublicKey, error) {
	resp, err := http.Get("https://api.workos.com/sso/jwks/" + wa.clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch WorkOS JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("WorkOS JWKS endpoint returned status %d", resp.StatusCode)
	}

	var jwks workosJWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode WorkOS JWKS: %w", err)
	}

	for _, k := range jwks.Keys {
		if k.Kid != kid || k.Kty != "RSA" {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return nil, fmt.Errorf("failed to decode WorkOS JWKS key N: %w", err)
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return nil, fmt.Errorf("failed to decode WorkOS JWKS key E: %w", err)
		}
		return &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: int(new(big.Int).SetBytes(eBytes).Int64()),
		}, nil
	}
	return nil, fmt.Errorf("WorkOS JWKS does not contain key %q", kid)
}

// userEmail calls the WorkOS User Management API to get the email of the user
// with the given WorkOS user ID.
func (wa *workosAuth) userEmail(userID string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.workos.com/user_management/users/"+userID, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+wa.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("WorkOS API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("WorkOS API returned status %d for user %q", resp.StatusCode, userID)
	}

	var user struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("failed to decode WorkOS user response: %w", err)
	}
	if user.Email == "" {
		return "", fmt.Errorf("WorkOS user %q has no email", userID)
	}
	return user.Email, nil
}
