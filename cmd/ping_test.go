package cmd

import (
	"net/http"
	"reanahub/reana-client-go/pkg/errorhandler"
	"testing"

	"github.com/spf13/viper"
)

var pingServerPath = "/api/you"

func TestPing(t *testing.T) {
	params := TestCmdParams{
		cmd: "ping",
		serverResponses: map[string]ServerResponse{
			pingServerPath: {
				statusCode:   http.StatusOK,
				responseFile: "ping.json",
			},
		},
		expected: []string{
			"REANA server version: 0.9.0a5",
			"Authenticated as: <john.doe@example.org>",
		},
	}
	testCmdRun(t, params)
}

func TestUnreachableServer(t *testing.T) {
	viper.Set("server-url", "https://unreachable.invalid")
	t.Cleanup(func() {
		viper.Reset()
	})

	rootCmd := NewRootCmd()
	output, err := ExecuteCommand(rootCmd, "ping", "-t", "1234")

	if err == nil {
		t.Errorf("Expected an error, instead got '%s'", output)
	}

	expectedErr := "'https://unreachable.invalid' not found, please verify the provided server URL or check your internet connection"
	if errorhandler.HandleApiError(err).Error() != expectedErr {
		t.Errorf("Expected server not found error, instead got '%s'", err.Error())
	}
}
