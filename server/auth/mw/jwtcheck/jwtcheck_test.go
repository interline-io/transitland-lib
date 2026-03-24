package jwtcheck

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/stretchr/testify/assert"
)

func testRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func signToken(t *testing.T, key *rsa.PrivateKey, claims jwt.Claims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	s, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestDiscoverJWKSURLWithContext(t *testing.T) {
	t.Run("valid discovery document", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/.well-known/openid-configuration", r.URL.Path)
			json.NewEncoder(w).Encode(map[string]string{
				"jwks_uri": "https://example.com/.well-known/jwks.json",
			})
		}))
		defer srv.Close()
		url, err := discoverJWKSURLWithContext(context.Background(), srv.URL)
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com/.well-known/jwks.json", url)
	})

	t.Run("missing jwks_uri", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]string{"issuer": "https://example.com"})
		}))
		defer srv.Close()
		_, err := discoverJWKSURLWithContext(context.Background(), srv.URL)
		assert.ErrorContains(t, err, "missing jwks_uri")
	})

	t.Run("non-200 response includes URL", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()
		_, err := discoverJWKSURLWithContext(context.Background(), srv.URL)
		assert.ErrorContains(t, err, "returned status 404")
		assert.ErrorContains(t, err, srv.URL)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not json"))
		}))
		defer srv.Close()
		_, err := discoverJWKSURLWithContext(context.Background(), srv.URL)
		assert.ErrorContains(t, err, "failed to parse")
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately
		_, err := discoverJWKSURLWithContext(ctx, "http://localhost:0")
		assert.Error(t, err)
	})
}

func TestValidateJwt(t *testing.T) {
	key := testRSAKey(t)
	audience := "test-audience"
	issuer := "test-issuer"
	keyFunc := func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return &key.PublicKey, nil
	}

	t.Run("valid token", func(t *testing.T) {
		tokenStr := signToken(t, key, &customClaims{
			Email: "user@example.com",
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "user-123",
				Audience:  jwt.ClaimStrings{audience},
				Issuer:    issuer,
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		})
		claims, err := validateJwt(keyFunc, audience, issuer, tokenStr)
		assert.NoError(t, err)
		assert.Equal(t, "user-123", claims.Subject)
		assert.Equal(t, "user@example.com", claims.Email)
	})

	t.Run("wrong audience", func(t *testing.T) {
		tokenStr := signToken(t, key, &customClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "user-123",
				Audience:  jwt.ClaimStrings{"wrong-audience"},
				Issuer:    issuer,
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		})
		_, err := validateJwt(keyFunc, audience, issuer, tokenStr)
		assert.Error(t, err)
	})

	t.Run("wrong issuer", func(t *testing.T) {
		tokenStr := signToken(t, key, &customClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "user-123",
				Audience:  jwt.ClaimStrings{audience},
				Issuer:    "wrong-issuer",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		})
		_, err := validateJwt(keyFunc, audience, issuer, tokenStr)
		assert.Error(t, err)
	})

	t.Run("expired token", func(t *testing.T) {
		tokenStr := signToken(t, key, &customClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "user-123",
				Audience:  jwt.ClaimStrings{audience},
				Issuer:    issuer,
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			},
		})
		_, err := validateJwt(keyFunc, audience, issuer, tokenStr)
		assert.Error(t, err)
	})

	t.Run("wrong signing key", func(t *testing.T) {
		otherKey := testRSAKey(t)
		tokenStr := signToken(t, otherKey, &customClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "user-123",
				Audience:  jwt.ClaimStrings{audience},
				Issuer:    issuer,
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		})
		_, err := validateJwt(keyFunc, audience, issuer, tokenStr)
		assert.Error(t, err)
	})
}

func TestNewJWTHandler(t *testing.T) {
	key := testRSAKey(t)
	audience := "test-audience"
	issuer := "test-issuer"
	keyFunc := func(token *jwt.Token) (any, error) {
		return &key.PublicKey, nil
	}

	validToken := func(sub, email string) string {
		return signToken(t, key, &customClaims{
			Email: email,
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   sub,
				Audience:  jwt.ClaimStrings{audience},
				Issuer:    issuer,
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		})
	}

	// Downstream handler that records the user from context
	captureUser := func() (http.Handler, func() authn.User) {
		var user authn.User
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user = authn.ForContext(r.Context())
			w.WriteHeader(http.StatusOK)
		})
		return h, func() authn.User { return user }
	}

	t.Run("valid token sets user context", func(t *testing.T) {
		next, getUser := captureUser()
		mw := newJWTHandler(keyFunc, audience, issuer, false)
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+validToken("user-123", "user@example.com"))
		rr := httptest.NewRecorder()
		mw(next).ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		u := getUser()
		assert.NotNil(t, u)
		assert.Equal(t, "user-123", u.ID())
	})

	t.Run("useEmailAsId uses email as ID", func(t *testing.T) {
		next, getUser := captureUser()
		mw := newJWTHandler(keyFunc, audience, issuer, true)
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+validToken("user-123", "user@example.com"))
		rr := httptest.NewRecorder()
		mw(next).ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		u := getUser()
		assert.NotNil(t, u)
		assert.Equal(t, "user@example.com", u.ID())
	})

	t.Run("no auth header passes through", func(t *testing.T) {
		next, getUser := captureUser()
		mw := newJWTHandler(keyFunc, audience, issuer, false)
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		mw(next).ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Nil(t, getUser())
	})

	t.Run("invalid token returns 401", func(t *testing.T) {
		next, _ := captureUser()
		mw := newJWTHandler(keyFunc, audience, issuer, false)
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.here")
		rr := httptest.NewRecorder()
		mw(next).ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("bearer prefix only passes through", func(t *testing.T) {
		next, getUser := captureUser()
		mw := newJWTHandler(keyFunc, audience, issuer, false)
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer ")
		rr := httptest.NewRecorder()
		mw(next).ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Nil(t, getUser())
	})
}
