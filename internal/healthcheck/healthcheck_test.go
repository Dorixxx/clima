package healthcheck

import (
	"testing"

	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
)

func TestUnauthorizedDisabledStatusMessage(t *testing.T) {
	t.Run("expired token gets explicit status", func(t *testing.T) {
		auth := &coreauth.Auth{
			LastError: &coreauth.Error{
				Message: "Provided authentication token is expired. Please try signing in again",
			},
		}
		got := unauthorizedDisabledStatusMessage(auth)
		if got != "disabled by health check: token expired, sign in again" {
			t.Fatalf("unexpected status message: %q", got)
		}
	})

	t.Run("invalidated token gets explicit status", func(t *testing.T) {
		auth := &coreauth.Auth{
			LastError: &coreauth.Error{
				Message: "Your authentication token has been invalidated. Please try signing in again",
			},
		}
		got := unauthorizedDisabledStatusMessage(auth)
		if got != "disabled by health check: token invalidated, sign in again" {
			t.Fatalf("unexpected status message: %q", got)
		}
	})
}
