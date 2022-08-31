package ping

import (
	"net/http"
	"reanahub/reana-client-go/cmd/internal"
	"reanahub/reana-client-go/pkg/errorhandler"
	"testing"

	"github.com/spf13/viper"
)

var pingServerPath = "/api/you"

func TestPing(t *testing.T) {
	params := internal.TestCmdParams{
		Cmd: NewCmd(),
		ServerResponses: map[string]internal.ServerResponse{
			pingServerPath: {
				StatusCode:   http.StatusOK,
				ResponseFile: "testdata/success.json",
			},
		},
		Expected: []string{
			"REANA server version: 0.9.0a5",
			"Authenticated as: <john.doe@example.org>",
		},
	}
	internal.TestCmdRun(t, params)
}

func TestUnreachableServer(t *testing.T) {
	viper.Set("server-url", "https://unreachable.invalid")
	t.Cleanup(func() {
		viper.Reset()
	})

	output, err := internal.ExecuteCommand(NewCmd(), "-t", "1234")

	if err == nil {
		t.Errorf("Expected an error, instead got '%s'", output)
	}

	expectedErr := "'https://unreachable.invalid' not found, please verify the provided server URL or check your internet connection"
	if errorhandler.HandleApiError(err).Error() != expectedErr {
		t.Errorf("Expected server not found error, instead got '%s'", err.Error())
	}
}
