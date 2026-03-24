package jwtcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	keyfunc "github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/auth/authn"
)

// JWTMiddleware checks and pulls user information from JWT in Authorization header.
// The JWT is validated against a static RSA public key loaded from pubKeyPath.
func JWTMiddleware(jwtAudience string, jwtIssuer string, pubKeyPath string, useEmailAsId bool) (func(http.Handler) http.Handler, error) {
	verifyBytes, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return nil, err
	}
	verifyKey, err := jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
	if err != nil {
		return nil, err
	}
	keyFunc := func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return verifyKey, nil
	}
	return newJWTHandler(keyFunc, jwtAudience, jwtIssuer, useEmailAsId), nil
}

// JWTMiddlewareOIDC checks and pulls user information from JWT in Authorization header.
// The JWKS keys are discovered from the issuer's OpenID Connect discovery endpoint.
func JWTMiddlewareOIDC(jwtAudience string, jwtIssuer string, useEmailAsId bool) (func(http.Handler) http.Handler, error) {
	jwksURL, err := discoverJWKSURL(jwtIssuer)
	if err != nil {
		return nil, fmt.Errorf("OIDC discovery failed: %w", err)
	}
	kf, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("failed to create JWKS keyfunc: %w", err)
	}
	return newJWTHandler(kf.Keyfunc, jwtAudience, jwtIssuer, useEmailAsId), nil
}

// JWTMiddlewareOIDCCtx is like JWTMiddlewareOIDC but accepts a context that controls
// the lifetime of the background JWKS refresh goroutine.
func JWTMiddlewareOIDCCtx(ctx context.Context, jwtAudience string, jwtIssuer string, useEmailAsId bool) (func(http.Handler) http.Handler, error) {
	jwksURL, err := discoverJWKSURL(jwtIssuer)
	if err != nil {
		return nil, fmt.Errorf("OIDC discovery failed: %w", err)
	}
	kf, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("failed to create JWKS keyfunc: %w", err)
	}
	return newJWTHandler(kf.Keyfunc, jwtAudience, jwtIssuer, useEmailAsId), nil
}

// discoverJWKSURL fetches the OIDC discovery document from the issuer and returns the jwks_uri.
func discoverJWKSURL(issuer string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return discoverJWKSURLWithContext(ctx, issuer)
}

func discoverJWKSURLWithContext(ctx context.Context, issuer string) (string, error) {
	discoveryURL := strings.TrimRight(issuer, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create OIDC discovery request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch OIDC discovery document from %s: %w", discoveryURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OIDC discovery endpoint returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read OIDC discovery response: %w", err)
	}
	var doc struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return "", fmt.Errorf("failed to parse OIDC discovery document: %w", err)
	}
	if doc.JWKSURI == "" {
		return "", errors.New("OIDC discovery document missing jwks_uri")
	}
	return doc.JWKSURI, nil
}

func newJWTHandler(keyFunc jwt.Keyfunc, jwtAudience string, jwtIssuer string, useEmailAsId bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tokenString, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok && tokenString != "" {
				claims, err := validateJwt(keyFunc, jwtAudience, jwtIssuer, tokenString)
				if err != nil {
					log.Error().Err(err).Msgf("invalid jwt token")
					writeJsonError(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
				if claims == nil {
					log.Error().Msgf("no claims")
					writeJsonError(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
				userId := claims.Subject
				if useEmailAsId {
					userId = claims.Email
				}
				jwtUser := authn.NewCtxUser(userId, claims.Subject, claims.Email).WithRoles("has_jwt")
				r = r.WithContext(authn.WithUser(r.Context(), jwtUser))
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CustomClaimsExample contains the JWT claims used by the middleware.
type CustomClaimsExample = customClaims

type customClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func validateJwt(keyFunc jwt.Keyfunc, jwtAudience string, jwtIssuer string, tokenString string) (*customClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&customClaims{},
		keyFunc,
		jwt.WithAudience(jwtAudience),
		jwt.WithIssuer(jwtIssuer),
	)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*customClaims)
	if !ok {
		return nil, errors.New("invalid claims type")
	}
	return claims, nil
}

func writeJsonError(w http.ResponseWriter, msg string, statusCode int) {
	a := map[string]string{
		"error": msg,
	}
	jj, _ := json.Marshal(&a)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(jj)
}
