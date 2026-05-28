// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package workos

import (
	"bytes"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	baseURL = "https://api.workos.com"

	// publicKeyTTL is an arbitrary TTL used to eventually evict cached public
	// keys and ensure they are re-fetched periodically. It does not reflect any
	// documented WorkOS key rotation policy.
	publicKeyTTL      = 24 * time.Hour
	publicKeyMaxCache = 100
)

var (
	ErrInvalidToken            = errors.New("WorkOS provided an invalid JWT token")
	errCannotRetrievePublicKey = errors.New("cannot retrieve the WorkOS public key")
)

type Workos struct {
	ClientID      string
	apiKey        string
	webhookSecret string
	actionsSecret string
	DevMode       bool
	keysMu        sync.RWMutex
	publicKeys    map[string]publicKey // kid → cached key
	transport     http.RoundTripper
}

// user holds the user information returned by WorkOS after token verification.
type user struct {
	ID        string
	Email     string
	FirstName string
	LastName  string
}

type publicKey struct {
	key       *rsa.PublicKey
	alg       string
	expiresAt time.Time
}

type claims struct {
	jwt.RegisteredClaims
	ClientID string `json:"client_id"`
	OrgID    string `json:"org_id"`
}

type jwks struct {
	Keys []struct {
		Kty string `json:"kty"`
		Use string `json:"use"`
		Alg string `json:"alg"`
		Kid string `json:"kid"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

func New(clientID, apiKey, webhookSecret, actionsSecret string, devMode bool) *Workos {
	return &Workos{
		ClientID:      clientID,
		apiKey:        apiKey,
		webhookSecret: webhookSecret,
		actionsSecret: actionsSecret,
		DevMode:       devMode,
		publicKeys:    make(map[string]publicKey),
		transport:     &http.Transport{Proxy: nil},
	}
}

// publicKey returns the RSA public key for the given key ID and algorithm,
// using the in-memory cache when available.
func (wo *Workos) publicKey(kid, alg string) (*rsa.PublicKey, error) {
	if kid == "" {
		return nil, fmt.Errorf("WorkOS provided a JWT with an empty key identifier (kid)")
	}
	wo.keysMu.RLock()
	if pk, ok := wo.publicKeys[kid]; ok && pk.alg == alg && time.Now().Before(pk.expiresAt) {
		wo.keysMu.RUnlock()
		return pk.key, nil
	}
	wo.keysMu.RUnlock()

	key, err := wo.fetchPublicKey(kid, alg)
	if err != nil {
		return nil, err
	}

	wo.keysMu.Lock()
	defer wo.keysMu.Unlock()

	now := time.Now()
	for k, e := range wo.publicKeys {
		if now.After(e.expiresAt) {
			delete(wo.publicKeys, k)
		}
	}

	if len(wo.publicKeys) < publicKeyMaxCache {
		wo.publicKeys[kid] = publicKey{
			key:       key,
			alg:       alg,
			expiresAt: time.Now().Add(publicKeyTTL),
		}
	}

	return key, nil
}

// fetchPublicKey fetches the WorkOS JWKS and returns the RSA public key for the
// given key ID and algorithm.
func (wo *Workos) fetchPublicKey(kid, alg string) (*rsa.PublicKey, error) {
	var jwks jwks
	err := wo.call(http.MethodGet, "/sso/jwks/"+url.PathEscape(wo.ClientID), http.StatusOK, nil, &jwks)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch WorkOS JWKS: %s", err)
	}
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			slog.Warn("workos: JWKS contains non-RSA key", "kid", k.Kid, "kty", k.Kty)
			if k.Kid == kid {
				return nil, fmt.Errorf("WorkOS JWKS key %q has unsupported type %q", kid, k.Kty)
			}
			continue
		}
		if k.Kid != kid {
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

// Authenticate verifies the WorkOS JWT and returns the authenticated user's
// information and their organization external ID.
func (wo *Workos) Authenticate(token string) (*user, uuid.UUID, error) {
	var claims claims

	parsed, err := jwt.ParseWithClaims(
		token,
		&claims,
		func(t *jwt.Token) (any, error) {
			_, isRSA := t.Method.(*jwt.SigningMethodRSA)
			_, isPSS := t.Method.(*jwt.SigningMethodRSAPSS)
			if !isRSA && !isPSS {
				return nil, fmt.Errorf("WorkOS signed the JWT using %s, but Krenalis only supports RSA and RSA-PSS signing methods", t.Method.Alg())
			}
			kid, _ := t.Header["kid"].(string)
			if kid == "" {
				return nil, fmt.Errorf("WorkOS provided a JWT with an empty key identifier (kid)")
			}
			key, err := wo.publicKey(kid, t.Method.Alg())
			if err != nil {
				return nil, fmt.Errorf("%w: %s", errCannotRetrievePublicKey, err)
			}
			return key, nil
		},
		jwt.WithExpirationRequired(),
	)

	if err != nil {
		if errors.Is(err, errCannotRetrievePublicKey) {
			return nil, uuid.UUID{}, err
		}
		return nil, uuid.UUID{}, ErrInvalidToken
	}

	if !parsed.Valid {
		return nil, uuid.UUID{}, ErrInvalidToken
	}

	if claims.ClientID != wo.ClientID {
		return nil, uuid.UUID{}, fmt.Errorf("JWT client_id does not match configured client ID")
	}
	if claims.Subject == "" {
		return nil, uuid.UUID{}, ErrInvalidToken
	}
	if claims.OrgID == "" {
		return nil, uuid.UUID{}, ErrInvalidToken
	}

	userID := claims.Subject

	var userRes struct {
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}

	err = wo.call(http.MethodGet, "/user_management/users/"+url.PathEscape(userID), http.StatusOK, nil, &userRes)
	if err != nil {
		return nil, uuid.UUID{}, fmt.Errorf("failed to fetch WorkOS user: %s", err)
	}

	organizationID, err := wo.organization(claims.OrgID)
	if err != nil {
		return nil, uuid.UUID{}, fmt.Errorf("cannot retrieve WorkOS organization: %s", err)
	}

	user := &user{
		ID:        userID,
		Email:     userRes.Email,
		FirstName: userRes.FirstName,
		LastName:  userRes.LastName,
	}

	return user, organizationID, nil
}

// organization fetches the WorkOS organization and returns its external ID as a
// UUID, which is the Krenalis-side organization identifier.
func (wo *Workos) organization(orgID string) (uuid.UUID, error) {
	if strings.TrimSpace(orgID) == "" {
		return uuid.UUID{}, fmt.Errorf("missing organization ID")
	}
	var orgRes struct {
		ExternalID string `json:"external_id"`
	}
	err := wo.call(http.MethodGet, "/organizations/"+url.PathEscape(orgID), http.StatusOK, nil, &orgRes)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to fetch WorkOS organization %s: %s", orgID, err)
	}
	if orgRes.ExternalID == "" {
		return uuid.UUID{}, fmt.Errorf("WorkOS organization %s has no external ID", orgID)
	}
	id, err := uuid.Parse(orgRes.ExternalID)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("WorkOS organization %s has invalid external ID", orgID)
	}
	return id, nil
}

// call executes an HTTP request to the WorkOS API and returns the HTTP status
// code.
func (wo *Workos) call(method, path string, expectedStatus int, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to encode request body: %s", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+wo.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := wo.transport.RoundTrip(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if out != nil {
		err := json.NewDecoder(resp.Body).Decode(out)
		if err != nil {
			return err
		}
	}

	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("WorkOS returned unexpected status %d", resp.StatusCode)
	}

	return nil
}

// VerifyActionSignature verifies the HMAC-SHA256 signature of an incoming
// WorkOS action request. sigHeader is the value of the "WorkOS-Signature"
// header, which has the format "t=<timestamp_ms>,v1=<hex_digest>". The signed
// message is "<timestamp>.<rawBody>".
func (wo *Workos) VerifyActionSignature(rawBody []byte, sigHeader string) error {
	return wo.verifyHMACSignature(rawBody, sigHeader, wo.actionsSecret)
}

// VerifyWebhookSignature verifies the HMAC-SHA256 signature of an incoming
// WorkOS webhook event. sigHeader is the value of the "WorkOS-Signature"
// header, which has the format "t=<timestamp_ms>,v1=<hex_digest>". The signed
// message is "<timestamp>.<rawBody>".
func (wo *Workos) VerifyWebhookSignature(rawBody []byte, sigHeader string) error {
	return wo.verifyHMACSignature(rawBody, sigHeader, wo.webhookSecret)
}

func (wo *Workos) verifyHMACSignature(rawBody []byte, sigHeader, secret string) error {
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

// BuildActionResponse builds and signs a WorkOS action response JSON payload.
// verdict must be "Allow" or "Deny"; errorMessage is included only on Deny.
func (wo *Workos) BuildActionResponse(verdict, errorMessage string) ([]byte, error) {
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
