package oauth2

import (
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	oauthURL    = "https://cp-1-dev.pacelink.net"
	oauthClient = "7d51282118633c3a7412d7456368ddfe172b7987d20b8e3e60ae18e8681fac61"
	oauthSecret = "141f891391d2b529bbf37b5ae5f57000f8b093956121db51c90fefb83930175c"

	// This token will expire in three years (Sep, 11, 2021) and belongs to the above
	// application and (wael@pace.car).
	activeToken = "85b7856f3055411c11b60f582fc091a624db4a38218abac2df9feb66bc6e7eb5"

	// Wael's Cockpit unique identifier.
	userID = "b773de39-93d8-4aa4-94a3-356900e55956"
)

func dummyHandler(w http.ResponseWriter, r *http.Request) {}

func TestMiddleware(t *testing.T) {
	var middleware = Middleware{
		URL:          oauthURL,
		ClientID:     oauthClient,
		ClientSecret: oauthSecret,
	}

	router := mux.NewRouter()
	router.Use(middleware.Handler)
	router.HandleFunc("/broken", dummyHandler)
	router.HandleFunc("/inactive", dummyHandler)

	// Test no token.
	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/broken", nil)
	router.ServeHTTP(rw, req)

	if rw.Body.String() != "Unauthorized\n" {
		t.Fatalf("Expected `Unauthorized` as body when *no* token is sent, got %s.", rw.Body)
	}

	// Test bad token.
	rw = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/broken", nil)
	req.Header.Set("Authorization", "Bearer sometoken")
	router.ServeHTTP(rw, req)

	if rw.Body.String() != "User token is invalid\n" {
		t.Fatalf("Expected `User token is invalid.` as body when *bad* token is sent, got %s.", rw.Body)
	}

	// Check for data we are interested in inside the context.
	testMiddlewareHandler := func(w http.ResponseWriter, r *http.Request) {
		// Check if we have the X-UID.
		if rw.Result().StatusCode != 200 || UserID(r.Context()) != userID {
			t.Fatal("Expected successful request and X-UID stored in request context.")
		}

		// Check if we have the token.
		receivedToken := BearerToken(r.Context())

		if receivedToken != activeToken {
			t.Fatalf("Expected %s, got: %s", activeToken, receivedToken)
		}

		// Check if we have the scopes.
		scopes := Scopes(r.Context())

		if len(scopes) < 2 {
			t.Fatal("Expected scopes: dtc:codes:read and dtc:codes:write, got nothing.")
		}

		if scopes[0] != "dtc:codes:read" || scopes[1] != "dtc:codes:write" {
			t.Fatalf("Expected scopes: dtc:codes:read and dtc:codes:write, got: %s", scopes)
		}

		// Check if we have the client ID.
		clientID := ClientID(r.Context())

		if clientID != oauthClient {
			t.Fatalf("Expected ClientID %s, got: %s", oauthClient, clientID)
		}
	}

	rw = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/working", nil)
	req.Header.Set("Authorization", "Bearer "+activeToken)
	router.HandleFunc("/working", testMiddlewareHandler)
	router.ServeHTTP(rw, req)

	// This is a last check to make sure everything is good. We must do this check,
	// because it indirectly ensures that the testMiddlewareHandler did actually
	// run. We do not have other options because our /introspect endpoint does not
	// differentiate between bad and old tokens.
	if rw.Result().StatusCode != 200 || rw.Body.String() == "Unauthorized\n" {
		t.Fatalf("Unexpected results using token: %s, perhaps it expired?", activeToken)
	}
}
