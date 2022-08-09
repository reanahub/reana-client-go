package cmd

import (
	"net/http"
	"net/http/httptest"
	"reanahub/reana-client-go/utils"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestPing(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if accessToken := r.URL.Query().Get("access_token"); accessToken != "1234" {
			t.Errorf("Expected access token '1234', got '%v'", accessToken)
		}
		if r.URL.Path == "/api/you" {
			w.Header().Add("Content-Type", "application/json")
			_, err := w.Write([]byte(`{
				"email": "john.doe@example.org",
				"reana_server_version": "0.9.0a5"
			}`))
			if err != nil {
				t.Fatalf("Error while writing response body: %v", err)
			}
		} else {
			t.Fatalf("Unexpected request to '%v'", r.URL.Path)
		}
	}))
	defer server.Close()

	viper.Set("server-url", server.URL)
	defer viper.Reset()

	rootCmd := NewRootCmd()
	output, err := utils.ExecuteCommand(rootCmd, "ping", "-t", "1234")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(output, "REANA server version: 0.9.0a5") {
		t.Error("Expected server version in command's output")
	}
	if !strings.Contains(output, "Authenticated as: <john.doe@example.org>") {
		t.Error("Expected user email in command's output")
	}
}
