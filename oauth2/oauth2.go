// Package oauth2 provides a middelware that introspects the auth token on
// behalf of PACE services and populate the request context with useful information
// when the token is valid, otherwise aborts the request.
//
// See example_usage.go for an example usage (pardon the runny wording).

// TODO
// table tests.
// Add copyright notice.
package oauth2

import (
	"context"
	"net/http"
	"strings"
)

type ctxkey string

var tokenKey = ctxkey("Token")

const headerPrefix = "Bearer "

// Oauth2 Middleware.
type Middleware struct {
	URL          string
	ClientID     string
	ClientSecret string
}

type token struct {
	value    string
	userID   string
	clientID string
	scopes   []string
}

// Should take token, introspect it, and put the token and other relevant information back
// in the context.
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		qualifiedToken := r.Header.Get("Authorization")

		items := strings.Split(qualifiedToken, "Bearer ")
		if len(items) < 2 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tokenValue := items[1]
		var s introspectResponse
		err := introspect(*m, tokenValue, &s)

		switch err {
		case errBadUpstreamResponse:
			http.Error(w, err.Error(), http.StatusBadGateway)
		case errUpstreamConnection:
			http.Error(w, err.Error(), http.StatusUnauthorized)
		case errInvalidToken:
			http.Error(w, err.Error(), http.StatusUnauthorized)
		}

		token := fromIntrospectResponse(s, tokenValue)

		ctx := context.WithValue(r.Context(), tokenKey, &token)
		next.ServeHTTP(w, r.WithContext(ctx))
		return
	})
}

func fromIntrospectResponse(s introspectResponse, tokenValue string) token {
	token := token{
		userID:   s.UserID,
		value:    tokenValue,
		clientID: s.ClientID,
	}

	if s.Scope != "" {
		scopes := strings.Split(s.Scope, " ")
		token.scopes = scopes
	}

	return token
}

// TODO Pseudoish. To test.
func Request(ctx context.Context, r *http.Request) *http.Request {
	token := BearerToken(ctx)
	authHeaderVal := headerPrefix + token
	r.Header.Set("Authorization: ", authHeaderVal)
	return r
}

func BearerToken(ctx context.Context) string {
	token := ctx.Value(tokenKey).(*token)
	return token.value
}

func HasScope(ctx context.Context, scope string) bool {
	token := ctx.Value(tokenKey).(*token)

	for _, v := range token.scopes {
		if v == scope {
			return true
		}
	}

	return false
}

func UserID(ctx context.Context) string {
	token := ctx.Value(tokenKey).(*token)

	return token.userID
}

func Scopes(ctx context.Context) []string {
	token := ctx.Value(tokenKey).(*token)

	return token.scopes
}

func ClientID(ctx context.Context) string {
	token := ctx.Value(tokenKey).(*token)

	return token.clientID
}
