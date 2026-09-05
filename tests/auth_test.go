package tests

import (
	"os"
	"testing"

	"github.com/agusibrahim/gpmc-go/internal/api"
)

func TestAuth(t *testing.T) {
	authData := os.Getenv("GP_AUTH_DATA")
	if authData == "" {
		t.Skip("GP_AUTH_DATA not set")
	}

	a, err := api.New(authData)
	if err != nil {
		t.Fatalf("api.New failed: %v", err)
	}

	token, err := a.BearerToken()
	if err != nil {
		t.Fatalf("BearerToken failed: %v", err)
	}

	t.Logf("Got BearerToken successfully: %s...", token[:30])
}
