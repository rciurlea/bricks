// Copyright © 2018 by PACE Telematics GmbH. All rights reserved.
// Created at 2018/09/03 by Mohamed Wael Khobalatte

package oauth2

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/pace/bricks/maintenance/log"
)

func Example() {
	r := mux.NewRouter()
	middleware := Middleware{}

	r.Use(middleware.Handler)

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		userid, _ := UserID(r.Context())
		log.Printf("AUDIT: User %s does something", userid)

		if HasScope(r.Context(), "dtc:codes:write") {
			fmt.Fprintf(w, "User has scope.")
			return
		}

		fmt.Fprintf(w, "Your client may not have the right scopes to see the secret code")
	})

	srv := &http.Server{
		Handler: r,
		Addr:    "127.0.0.1:8000",
	}

	log.Fatal(srv.ListenAndServe())
}

func TestRequest(t *testing.T) {
	var to = token{
		value:    "somevalue",
		userID:   "someuserid",
		clientID: "someclientid",
		scope:    Scope("scope1 scope2"),
	}

	r := httptest.NewRequest("GET", "http://example.com", nil)
	ctx := context.WithValue(r.Context(), tokenKey, &to)
	r = r.WithContext(ctx)

	r2 := Request(r)
	header := r2.Header.Get("Authorization")

	if header != "Bearer somevalue" {
		t.Fatalf("Expected request to have authorization header, got: %v", header)
	}
}

func TestRequestWithNoToken(t *testing.T) {
	r := httptest.NewRequest("GET", "http://example.com", nil)
	r2 := Request(r)
	header := r2.Header.Get("Authorization")

	if header != "" {
		t.Fatalf("Expected request to have no authorization header, got: %v", header)
	}
}

func TestSuccessfulAccessors(t *testing.T) {
	expectedBearerToken := "somevalue"
	expectedUserID := "someuserid"
	expectedClientID := "someclientid"
	expectedScopes := Scope("scope1 scope2")

	var to = token{
		value:    expectedBearerToken,
		userID:   expectedUserID,
		clientID: expectedClientID,
		scope:    expectedScopes,
	}

	ctx := context.WithValue(context.TODO(), tokenKey, &to)

	uid, _ := UserID(ctx)
	cid, _ := ClientID(ctx)
	bearerToken, _ := BearerToken(ctx)
	scopes := Scopes(ctx)
	hasScope := HasScope(ctx, "scope2")

	if uid != expectedUserID {
		t.Fatalf("Expected %v, got: %v", expectedUserID, uid)
	}

	if cid != expectedClientID {
		t.Fatalf("Expected %v, got: %v", expectedClientID, cid)
	}

	if bearerToken != expectedBearerToken {
		t.Fatalf("Expected %v, got: %v", expectedBearerToken, bearerToken)
	}

	if scopes[0] != "scope1" || scopes[1] != "scope2" {
		t.Fatalf("Expected %v, got: %v", expectedScopes, scopes)
	}

	if !hasScope {
		t.Fatalf("Expected true, got: %v", hasScope)
	}
}

// Ensure we return sensible results when no data is present, and not panic.
func TestUnsucessfulAccessors(t *testing.T) {
	ctx := context.TODO()

	uid, uidOK := UserID(ctx)
	cid, cidOK := ClientID(ctx)
	bt, btOK := BearerToken(ctx)
	scopes := Scopes(ctx)
	hasScope := HasScope(ctx, "scope2")

	if uid != "" || uidOK {
		t.Fatalf("Expected no %v, got: %v", "UserID", uid)
	}

	if cid != "" || cidOK {
		t.Fatalf("Expected no %v, got: %v", "ClientID", cid)
	}

	if bt != "" || btOK {
		t.Fatalf("Expected no %v, got: %v", "BearerToken", bt)
	}

	if len(scopes) > 0 {
		t.Fatalf("Expected no scopes, got: %v", scopes)
	}

	if hasScope {
		t.Fatalf("Expected hasScope to return false, got: %v", hasScope)
	}
}
