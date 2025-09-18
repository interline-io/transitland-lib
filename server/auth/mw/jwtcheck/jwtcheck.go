package jwtcheck

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/form3tech-oss/jwt-go"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/auth/authn"
)

// JWTMiddleware checks and pulls user information from JWT in Authorization header.
func JWTMiddleware(jwtAudience string, jwtIssuer string, pubKeyPath string, useEmailAsId bool) (func(http.Handler) http.Handler, error) {
	var verifyKey *rsa.PublicKey
	verifyBytes, err := ioutil.ReadFile(pubKeyPath)
	if err != nil {
		return nil, err
	}
	verifyKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
	if err != nil {
		return nil, err
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tokenString := strings.Split(r.Header.Get("Authorization"), "Bearer "); len(tokenString) == 2 {
				claims, err := validateJwt(verifyKey, jwtAudience, jwtIssuer, tokenString[1])
				if err != nil {
					log.Error().Err(err).Msgf("invalid jwt token")
					writeJsonError(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
				if claims == nil {
					log.Error().Err(err).Msgf("no claims")
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
	}, nil
}

type CustomClaimsExample struct {
	Email string
	jwt.StandardClaims
}

func (c *CustomClaimsExample) Valid() error {
	return nil
}

func validateJwt(rsaPublicKey *rsa.PublicKey, jwtAudience string, jwtIssuer string, tokenString string) (*CustomClaimsExample, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaimsExample{}, func(token *jwt.Token) (interface{}, error) {
		return rsaPublicKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims := token.Claims.(*CustomClaimsExample)
	if !claims.VerifyAudience(jwtAudience, true) {
		return nil, errors.New("invalid audience")
	}
	if !claims.VerifyIssuer(jwtIssuer, true) {
		return nil, errors.New("invalid issuer")
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
