package cmd

import (
	"strings"
	"testing"

	"github.com/optikklabs/optikk/internal/endpoint"
)

// The approval page lives on the web app. Building it from the API base is the
// regression this guards: api.optikk.in serves the device routes as POST JSON
// only, so an API-derived URL 404s and login cannot complete.
func TestDeviceVerifyURLPointsAtTheWebApp(t *testing.T) {
	got := deviceVerifyURL("XL6D-V4V9")

	if want := endpoint.AppURL + "/device?user_code=XL6D-V4V9"; got != want {
		t.Errorf("deviceVerifyURL = %q, want %q", got, want)
	}
	if strings.Contains(got, endpoint.APIURL) {
		t.Errorf("device approval URL must not be built from the API base: %q", got)
	}
}

func TestDeviceVerifyURLEscapesTheUserCode(t *testing.T) {
	got := deviceVerifyURL("A B&C=D")

	if want := endpoint.AppURL + "/device?user_code=A+B%26C%3DD"; got != want {
		t.Errorf("deviceVerifyURL = %q, want %q", got, want)
	}
}
