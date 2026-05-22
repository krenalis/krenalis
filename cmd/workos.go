// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const workosBaseURL = "https://api.workos.com"

type workos struct {
	clientID      string
	apiKey        string
	webhookSecret string
	actionsSecret string
	devMode       bool
}

// workosUser holds the user information returned by WorkOS after token
// verification.
type workosUser struct {
	ID        string
	Email     string
	FirstName string
	LastName  string
}

type workosClaims struct {
	jwt.RegisteredClaims
	ClientID string `json:"client_id"`
	OrgID    string `json:"org_id"`
}

type workosJWKS struct {
	Keys []struct {
		Kty string `json:"kty"`
		Use string `json:"use"`
		Alg string `json:"alg"`
		Kid string `json:"kid"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

func NewWorkOS(clientID, apiKey, webhookSecret, actionsSecret string, devMode bool) *workos {
	return &workos{
		clientID:      clientID,
		apiKey:        apiKey,
		webhookSecret: webhookSecret,
		actionsSecret: actionsSecret,
		devMode:       devMode,
	}
}

// publicKey fetches the WorkOS JWKS and returns the RSA public key for the
// given key ID, supporting the given alg.
func (wo *workos) publicKey(kid, alg string) (*rsa.PublicKey, error) {
	var jwks workosJWKS
	status, err := wo.call(http.MethodGet, "/sso/jwks/"+wo.clientID, nil, &jwks)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch WorkOS JWKS: %s", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("WorkOS JWKS endpoint returned status %d", status)
	}
	for _, k := range jwks.Keys {
		if k.Kid != kid || k.Kty != "RSA" {
			continue
		}
		if k.Use != "" && k.Use != "sig" {
			continue
		}
		if k.Alg != alg {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return nil, fmt.Errorf("failed to decode WorkOS JWKS key N: %s", err)
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return nil, fmt.Errorf("failed to decode WorkOS JWKS key E: %s", err)
		}
		return &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: int(new(big.Int).SetBytes(eBytes).Int64()),
		}, nil
	}
	return nil, fmt.Errorf("WorkOS JWKS does not contain key %q", kid)
}

// verifyToken verifies the WorkOS JWT and returns the authenticated user's
// information and their organization external ID.
func (wo *workos) verifyToken(token string) (*workosUser, *uuid.UUID, error) {
	var claims workosClaims

	parsed, err := jwt.ParseWithClaims(
		token,
		&claims,
		func(t *jwt.Token) (any, error) {
			_, isRSA := t.Method.(*jwt.SigningMethodRSA)
			_, isPSS := t.Method.(*jwt.SigningMethodRSAPSS)
			if !isRSA && !isPSS {
				return nil, fmt.Errorf("unexpected JWT signing method: %v", t.Method.Alg())
			}
			kid, _ := t.Header["kid"].(string)
			if kid == "" {
				return nil, fmt.Errorf("JWT is missing kid header")
			}
			return wo.publicKey(kid, t.Method.Alg())
		},
		jwt.WithExpirationRequired(),
	)
	if err != nil || !parsed.Valid {
		return nil, nil, fmt.Errorf("invalid JWT")
	}

	if claims.ClientID != wo.clientID {
		return nil, nil, fmt.Errorf("JWT client_id does not match configured client ID")
	}
	if claims.Subject == "" {
		return nil, nil, fmt.Errorf("missing sub claim in JWT")
	}
	if claims.OrgID == "" {
		return nil, nil, fmt.Errorf("missing organization ID in JWT")
	}

	userID := claims.Subject

	var userRes struct {
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}

	status, err := wo.call(http.MethodGet, "/user_management/users/"+url.PathEscape(userID), nil, &userRes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch WorkOS token: %s", err)
	}
	if status != http.StatusOK {
		return nil, nil, fmt.Errorf("WorkOS API returned status %d for user %s", status, userID)
	}

	organizationID, err := wo.workosOrganization(claims.OrgID)
	if err != nil {
		return nil, nil, err
	}

	return &workosUser{ID: userID, Email: userRes.Email, FirstName: userRes.FirstName, LastName: userRes.LastName}, organizationID, nil
}

// workosOrganization fetches the WorkOS organization and returns its external
// ID as a UUID, which is the Krenalis-side organization identifier.
func (wo *workos) workosOrganization(orgID string) (*uuid.UUID, error) {
	var orgRes struct {
		ExternalID string `json:"external_id"`
	}
	status, err := wo.call(http.MethodGet, "/organizations/"+url.PathEscape(orgID), nil, &orgRes)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch WorkOS organization %s: %s", orgID, err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("WorkOS API returned status %d for organization %s", status, orgID)
	}
	if orgRes.ExternalID == "" {
		return nil, fmt.Errorf("WorkOS organization %s has no external ID", orgID)
	}
	id, err := uuid.Parse(orgRes.ExternalID)
	if err != nil {
		return nil, fmt.Errorf("WorkOS organization %s has invalid external ID", orgID)
	}
	return &id, nil
}

// call executes an HTTP request to the WorkOS API and returns the HTTP status
// code.
func (wo *workos) call(method, path string, body any, out any) (int, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, fmt.Errorf("failed to encode request body: %s", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, workosBaseURL+path, bodyReader)
	if err != nil {
		return 0, err
	}
	if wo.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+wo.apiKey)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("%s %s failed: %s", method, path, err)
	}
	defer resp.Body.Close()

	if out != nil {
		err := json.NewDecoder(resp.Body).Decode(out)
		if err != nil {
			return resp.StatusCode, fmt.Errorf("failed to decode response from %s %s: %s", method, path, err)
		}
	}
	return resp.StatusCode, nil
}

// verifyActionSignature verifies the HMAC-SHA256 signature of an incoming
// WorkOS action request. sigHeader is the value of the "WorkOS-Signature"
// header, which has the format "t=<timestamp_ms>,v1=<hex_digest>". The signed
// message is "<timestamp>.<rawBody>".
func (wo *workos) verifyActionSignature(rawBody []byte, sigHeader string) error {
	return wo.verifyHMACSignature(rawBody, sigHeader, wo.actionsSecret)
}

// verifyWebhookSignature verifies the HMAC-SHA256 signature of an incoming
// WorkOS webhook event. sigHeader is the value of the "WorkOS-Signature"
// header, which has the format "t=<timestamp_ms>,v1=<hex_digest>". The signed
// message is "<timestamp>.<rawBody>".
func (wo *workos) verifyWebhookSignature(rawBody []byte, sigHeader string) error {
	return wo.verifyHMACSignature(rawBody, sigHeader, wo.webhookSecret)
}

func (wo *workos) verifyHMACSignature(rawBody []byte, sigHeader, secret string) error {
	parts := strings.Split(sigHeader, ",")
	if len(parts) < 2 {
		return fmt.Errorf("invalid WorkOS webhook")
	}

	timestamp, ok := strings.CutPrefix(strings.TrimSpace(parts[0]), "t=")
	if !ok {
		return fmt.Errorf("invalid WorkOS webhook")
	}

	signature, ok := strings.CutPrefix(strings.TrimSpace(parts[1]), "v1=")
	if !ok {
		return fmt.Errorf("invalid WorkOS webhook")
	}

	if timestamp == "" || signature == "" {
		return fmt.Errorf("invalid WorkOS webhook")
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp in WorkOS-Signature header")
	}
	diff := time.Now().UnixMilli() - ts
	if diff < 0 || diff > 5*60*1000 {
		return fmt.Errorf("WorkOS signature timestamp is too old or in the future")
	}

	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("invalid hex in WorkOS-Signature header: %s", err)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(rawBody)
	if !hmac.Equal(mac.Sum(nil), sigBytes) {
		return fmt.Errorf("WorkOS signature mismatch")
	}

	return nil
}

// buildActionResponse builds and signs a WorkOS action response JSON payload.
// verdict must be "Allow" or "Deny"; errorMessage is included only on Deny.
func (wo *workos) buildActionResponse(verdict, errorMessage string) ([]byte, error) {
	type actionPayload struct {
		Timestamp    int64  `json:"timestamp"`
		Verdict      string `json:"verdict"`
		ErrorMessage string `json:"error_message,omitempty"`
	}
	t := time.Now().UnixMilli()
	p := actionPayload{Timestamp: t, Verdict: verdict, ErrorMessage: errorMessage}
	pJSON, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, []byte(wo.actionsSecret))
	fmt.Fprintf(mac, "%d.", t)
	mac.Write(pJSON)
	sig := hex.EncodeToString(mac.Sum(nil))
	type response struct {
		Object    string        `json:"object"`
		Payload   actionPayload `json:"payload"`
		Signature string        `json:"signature"`
	}
	return json.Marshal(
		response{
			Object:    "user_registration_action_response",
			Payload:   p,
			Signature: sig,
		},
	)
}
