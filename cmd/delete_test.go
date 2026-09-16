/*
This file is part of REANA.
Copyright (C) 2022, 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/reanahub/reana-client-go/client/operations"
	"github.com/reanahub/reana-client-go/pkg/config"

	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
)

type deleteAPIRequest struct {
	method string
	path   string
	query  url.Values
	body   operations.SetWorkflowStatusBody
}

type deleteAPIResponse struct {
	statusCode int
	body       string
}

type deleteAPIRun struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

func deleteRun(id, name, status string) deleteAPIRun {
	return deleteAPIRun{ID: id, Name: name, Status: status}
}

func deleteStatusResponse(id, name, status string) deleteAPIResponse {
	return deleteJSONResponse(http.StatusOK, deleteRun(id, name, status))
}

func deleteListResponse(runs ...deleteAPIRun) deleteAPIResponse {
	return deleteListPageResponse(len(runs), runs...)
}

func deleteListPageResponse(total int, runs ...deleteAPIRun) deleteAPIResponse {
	return deleteJSONResponse(http.StatusOK, struct {
		Items []deleteAPIRun `json:"items"`
		Total int            `json:"total"`
	}{
		Items: runs,
		Total: total,
	})
}

func deleteSuccessResponse() deleteAPIResponse {
	return deleteJSONResponse(http.StatusOK, map[string]string{
		"message": "Workflow successfully deleted",
		"status":  "deleted",
	})
}

func deleteFailureResponse(message string) deleteAPIResponse {
	return deleteJSONResponse(
		http.StatusInternalServerError,
		map[string]string{"message": message},
	)
}

func deleteJSONResponse(statusCode int, value any) deleteAPIResponse {
	body, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("Could not encode delete API response: %v", err))
	}
	return deleteAPIResponse{statusCode: statusCode, body: string(body)}
}

func executeDeleteCommand(
	t *testing.T,
	args []string,
	respond func(deleteAPIRequest) deleteAPIResponse,
) (string, []deleteAPIRequest, error) {
	t.Helper()

	var mu sync.Mutex
	requests := make([]deleteAPIRequest, 0)
	server := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			request := deleteAPIRequest{
				method: r.Method,
				path:   r.URL.Path,
				query:  r.URL.Query(),
			}
			if r.Method == http.MethodPut {
				if err := json.NewDecoder(r.Body).Decode(&request.body); err != nil {
					t.Errorf("Could not decode delete request body: %v", err)
				}
			}
			if got := r.Header.Get("Authorization"); got != "Bearer 1234" {
				t.Errorf("Expected Authorization Bearer 1234, got %q", got)
			}
			if request.query.Has("access_token") {
				t.Errorf("Access token leaked into query: %s", r.URL.RawQuery)
			}

			mu.Lock()
			requests = append(requests, request)
			mu.Unlock()

			response := respond(request)
			statusCode := response.statusCode
			if statusCode == 0 {
				statusCode = http.StatusOK
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)
			if _, err := w.Write([]byte(response.body)); err != nil {
				t.Errorf("Could not write delete API response: %v", err)
			}
		},
	))
	defer server.Close()

	viper.Set("server-url", server.URL)
	defer viper.Reset()

	rootCmd := NewRootCmd()
	commandArgs := append([]string{"delete", "-t", "1234"}, args...)
	output, err := ExecuteCommand(rootCmd, commandArgs...)

	mu.Lock()
	defer mu.Unlock()
	return output, append([]deleteAPIRequest(nil), requests...), err
}

func TestDeleteStandaloneRun(t *testing.T) {
	output, requests, err := executeDeleteCommand(
		t,
		[]string{"-w", "my_workflow"},
		func(request deleteAPIRequest) deleteAPIResponse {
			switch request.method + " " + request.path {
			case "GET /api/workflows/my_workflow/status":
				return deleteStatusResponse(
					"run-10",
					"my_workflow.10",
					"finished",
				)
			case "GET /api/workflows":
				return deleteListResponse(
					deleteRun("run-10", "my_workflow.10", "finished"),
				)
			case "PUT /api/workflows/my_workflow.10/status":
				return deleteSuccessResponse()
			default:
				t.Errorf(
					"Unexpected request: %s %s",
					request.method,
					request.path,
				)
				return deleteAPIResponse{statusCode: http.StatusNotFound}
			}
		},
	)
	if err != nil {
		t.Fatalf("Got unexpected error: %v", err)
	}
	if !strings.Contains(output, "my_workflow.10 has been deleted") {
		t.Errorf("Unexpected output: %q", output)
	}
	if len(requests) != 3 {
		t.Fatalf("Got %d API requests, want 3", len(requests))
	}

	listRequest := requests[1]
	if listRequest.method != http.MethodGet ||
		listRequest.path != "/api/workflows" {
		t.Fatalf(
			"Second request is %s %s, want GET /api/workflows",
			listRequest.method,
			listRequest.path,
		)
	}
	for key, want := range map[string]string{
		"page":                "1",
		"size":                "1000",
		"type":                "batch",
		"workflow_id_or_name": "my_workflow",
	} {
		if got := listRequest.query.Get(key); got != want {
			t.Errorf("List query %q = %q, want %q", key, got, want)
		}
	}
	gotStatus := strings.Split(listRequest.query.Get("status"), ",")
	if got, want := gotStatus,
		config.GetRunStatuses(false); !slices.Equal(got, want) {
		t.Errorf("List status filters = %v, want %v", got, want)
	}

	deleteRequest := requests[2]
	if got := deleteRequest.query.Get("status"); got != "deleted" {
		t.Errorf("Delete status = %q, want deleted", got)
	}
	if !deleteRequest.body.Workspace || deleteRequest.body.AllRuns {
		t.Errorf("Unexpected delete body: %+v", deleteRequest.body)
	}
}

func TestDeleteAllRunsTakesPrecedence(t *testing.T) {
	output, requests, err := executeDeleteCommand(
		t,
		[]string{
			"-w", "my_workflow.10",
			"--include-all-runs",
			"--include-all-restarts",
		},
		func(request deleteAPIRequest) deleteAPIResponse {
			if request.method != http.MethodPut ||
				request.path != "/api/workflows/my_workflow.10/status" {
				t.Errorf(
					"Unexpected request: %s %s",
					request.method,
					request.path,
				)
				return deleteAPIResponse{statusCode: http.StatusNotFound}
			}
			return deleteSuccessResponse()
		},
	)
	if err != nil {
		t.Fatalf("Got unexpected error: %v", err)
	}
	if !strings.Contains(
		output,
		"All workflows named 'my_workflow' have been deleted",
	) {
		t.Errorf("Unexpected output: %q", output)
	}
	if len(requests) != 1 {
		t.Fatalf("Got %d API requests, want 1", len(requests))
	}
	if !requests[0].body.Workspace || !requests[0].body.AllRuns {
		t.Errorf("Unexpected delete body: %+v", requests[0].body)
	}
}

func TestDeleteRestartSeriesChecksEveryPage(t *testing.T) {
	firstPage := make([]deleteAPIRun, restartSeriesPageSize)
	for i := range firstPage {
		majorRun := i + 100
		firstPage[i] = deleteRun(
			fmt.Sprintf("run-%d", majorRun),
			fmt.Sprintf("analysis.%d", majorRun),
			"finished",
		)
	}
	totalRuns := len(firstPage) + 2

	_, requests, err := executeDeleteCommand(
		t,
		[]string{"-w", "analysis.7"},
		func(request deleteAPIRequest) deleteAPIResponse {
			switch request.method + " " + request.path {
			case "GET /api/workflows/analysis.7/status":
				return deleteStatusResponse("run-7", "analysis.7", "finished")
			case "GET /api/workflows":
				switch request.query.Get("page") {
				case "1":
					return deleteListPageResponse(totalRuns, firstPage...)
				case "2":
					return deleteListPageResponse(
						totalRuns,
						deleteRun("run-7", "analysis.7", "finished"),
						deleteRun("run-7-1", "analysis.7.1", "failed"),
					)
				default:
					t.Errorf("Unexpected workflow-list page: %s", request.query)
					return deleteAPIResponse{statusCode: http.StatusNotFound}
				}
			default:
				t.Errorf(
					"Unexpected request: %s %s",
					request.method,
					request.path,
				)
				return deleteAPIResponse{statusCode: http.StatusNotFound}
			}
		},
	)
	if err == nil {
		t.Fatal("Expected restart-series deletion to fail")
	}
	if !strings.Contains(err.Error(), "Related runs: #7, #7.1") {
		t.Errorf("Expected second-page restart in error, got %q", err)
	}
	if len(requests) != 3 {
		t.Errorf("Got %d API requests, want 3", len(requests))
	}
}

func TestDeleteRestartSeriesFailsOnIncompletePagination(t *testing.T) {
	_, requests, err := executeDeleteCommand(
		t,
		[]string{"-w", "analysis.7"},
		func(request deleteAPIRequest) deleteAPIResponse {
			switch request.method + " " + request.path {
			case "GET /api/workflows/analysis.7/status":
				return deleteStatusResponse("run-7", "analysis.7", "finished")
			case "GET /api/workflows":
				switch request.query.Get("page") {
				case "1":
					return deleteListPageResponse(
						2,
						deleteRun("run-7", "analysis.7", "finished"),
					)
				case "2":
					return deleteListPageResponse(2)
				default:
					t.Errorf("Unexpected workflow-list page: %s", request.query)
					return deleteAPIResponse{statusCode: http.StatusNotFound}
				}
			default:
				t.Errorf(
					"Unexpected request: %s %s",
					request.method,
					request.path,
				)
				return deleteAPIResponse{statusCode: http.StatusNotFound}
			}
		},
	)
	if err == nil {
		t.Fatal("Expected incomplete pagination to fail")
	}
	for _, want := range []string{
		"cannot safely delete workflow run 'analysis.7'",
		"restart-series check received 1 of 2 workflow runs",
		"Nothing was deleted",
		"Retry the command",
		"contact your REANA administrator",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Expected error to contain %q, got %q", want, err)
		}
	}
	if len(requests) != 3 {
		t.Fatalf("Got %d API requests, want 3", len(requests))
	}
	for _, request := range requests {
		if request.method == http.MethodPut {
			t.Errorf(
				"Deletion attempted after incomplete pagination: %+v",
				request,
			)
		}
	}
}

func TestDeleteRestartSeriesRequiresExplicitOption(t *testing.T) {
	_, requests, err := executeDeleteCommand(
		t,
		[]string{"-w", "analysis.7.1"},
		func(request deleteAPIRequest) deleteAPIResponse {
			switch request.method + " " + request.path {
			case "GET /api/workflows/analysis.7.1/status":
				return deleteStatusResponse(
					"run-7-1",
					"analysis.7.1",
					"finished",
				)
			case "GET /api/workflows":
				return deleteListResponse(
					deleteRun("run-7-1", "analysis.7.1", "finished"),
					deleteRun("run-7-2", "analysis.7.2", "failed"),
					deleteRun("run-8", "analysis.8", "finished"),
					deleteRun("other", "other.7.2", "finished"),
				)
			default:
				t.Errorf(
					"Unexpected request: %s %s",
					request.method,
					request.path,
				)
				return deleteAPIResponse{statusCode: http.StatusNotFound}
			}
		},
	)
	if err == nil {
		t.Fatal("Expected restart-series deletion to fail")
	}
	for _, want := range []string{
		"part of a restart series",
		"Related runs: #7.1, #7.2",
		"--include-all-restarts",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Expected error to contain %q, got %q", want, err)
		}
	}
	if strings.Contains(err.Error(), "#8") {
		t.Errorf("Error contains unrelated runs: %q", err)
	}
	if len(requests) != 2 {
		t.Errorf("Got %d API requests, want 2", len(requests))
	}
}

func TestDeleteCompletedRestartSeriesByUUID(t *testing.T) {
	output, requests, err := executeDeleteCommand(
		t,
		[]string{
			"-w", "86f28b84-d59d-43ed-a8dd-7b4dada3aaa0",
			"--include-all-restarts",
		},
		func(request deleteAPIRequest) deleteAPIResponse {
			switch request.method + " " + request.path {
			case "GET /api/workflows/86f28b84-d59d-43ed-a8dd-7b4dada3aaa0/status":
				return deleteStatusResponse(
					"run-7-1",
					"physics.analysis.7.1",
					"finished",
				)
			case "GET /api/workflows":
				return deleteListResponse(
					deleteRun("run-7-1", "physics.analysis.7.1", "finished"),
					deleteRun("run-7-2", "physics.analysis.7.2", "failed"),
					deleteRun("run-7-3", "physics.analysis.7.3", "stopped"),
					deleteRun("run-8", "physics.analysis.8", "finished"),
				)
			case "PUT /api/workflows/run-7-1/status",
				"PUT /api/workflows/run-7-2/status",
				"PUT /api/workflows/run-7-3/status":
				return deleteSuccessResponse()
			default:
				t.Errorf(
					"Unexpected request: %s %s",
					request.method,
					request.path,
				)
				return deleteAPIResponse{statusCode: http.StatusNotFound}
			}
		},
	)
	if err != nil {
		t.Fatalf("Got unexpected error: %v", err)
	}
	for _, want := range []string{
		"Workflow run 'physics.analysis.7.1' including its restarts has been deleted",
		"#7.1, #7.2, #7.3",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("Expected output to contain %q, got %q", want, output)
		}
	}
	if len(requests) != 5 {
		t.Fatalf("Got %d API requests, want 5", len(requests))
	}
	for i, request := range requests[2:] {
		if request.body.AllRuns {
			t.Errorf("Delete request %d unexpectedly deletes all runs", i+1)
		}
		wantWorkspace := i == 0
		if request.body.Workspace != wantWorkspace {
			t.Errorf(
				"Delete request %d workspace = %t, want %t",
				i+1,
				request.body.Workspace,
				wantWorkspace,
			)
		}
	}
}

func TestDeleteRestartSeriesRejectsActiveRuns(t *testing.T) {
	_, requests, err := executeDeleteCommand(
		t,
		[]string{"-w", "analysis.7", "--include-all-restarts"},
		func(request deleteAPIRequest) deleteAPIResponse {
			switch request.method + " " + request.path {
			case "GET /api/workflows/analysis.7/status":
				return deleteStatusResponse("run-7", "analysis.7", "finished")
			case "GET /api/workflows":
				return deleteListResponse(
					deleteRun("run-7", "analysis.7", "finished"),
					deleteRun("run-7-1", "analysis.7.1", "running"),
					deleteRun("run-7-2", "analysis.7.2", "failed"),
				)
			default:
				t.Errorf(
					"Unexpected request: %s %s",
					request.method,
					request.path,
				)
				return deleteAPIResponse{statusCode: http.StatusNotFound}
			}
		},
	)
	if err == nil {
		t.Fatal("Expected active restart-series deletion to fail")
	}
	for _, want := range []string{
		"related restarts are still active",
		"Active runs: #7.1",
		"stop them first",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Expected error to contain %q, got %q", want, err)
		}
	}
	if strings.Contains(err.Error(), "#7.2") {
		t.Errorf("Error reports a completed run as active: %q", err)
	}
	if len(requests) != 2 {
		t.Errorf("Got %d API requests, want 2", len(requests))
	}
}

func TestDeleteAlreadyDeletedRun(t *testing.T) {
	output, requests, err := executeDeleteCommand(
		t,
		[]string{"-w", "analysis.7.1"},
		func(request deleteAPIRequest) deleteAPIResponse {
			if request.method != http.MethodGet ||
				request.path != "/api/workflows/analysis.7.1/status" {
				t.Errorf(
					"Unexpected request: %s %s",
					request.method,
					request.path,
				)
				return deleteAPIResponse{statusCode: http.StatusNotFound}
			}
			return deleteStatusResponse("run-7-1", "analysis.7.1", "deleted")
		},
	)
	if err != nil {
		t.Fatalf("Got unexpected error: %v", err)
	}
	if !strings.Contains(
		output,
		"Workflow run 'analysis.7.1' is already deleted.",
	) {
		t.Errorf("Unexpected output: %q", output)
	}
	if len(requests) != 1 {
		t.Errorf("Got %d API requests, want 1", len(requests))
	}
}

func TestDeleteRestartSeriesAggregatesFailures(t *testing.T) {
	output, requests, err := executeDeleteCommand(
		t,
		[]string{"-w", "analysis.7", "--include-all-restarts"},
		func(request deleteAPIRequest) deleteAPIResponse {
			switch request.method + " " + request.path {
			case "GET /api/workflows/analysis.7/status":
				return deleteStatusResponse("run-7", "analysis.7", "finished")
			case "GET /api/workflows":
				return deleteListResponse(
					deleteRun("run-7", "analysis.7", "finished"),
					deleteRun("run-7-1", "analysis.7.1", "failed"),
					deleteRun("run-7-2", "analysis.7.2", "stopped"),
					deleteRun("run-7-3", "analysis.7.3", "finished"),
				)
			case "PUT /api/workflows/run-7/status":
				return deleteSuccessResponse()
			case "PUT /api/workflows/run-7-1/status":
				return deleteFailureResponse("first sibling deletion failed")
			case "PUT /api/workflows/run-7-2/status":
				return deleteSuccessResponse()
			case "PUT /api/workflows/run-7-3/status":
				return deleteFailureResponse("last deletion failed")
			default:
				t.Errorf(
					"Unexpected request: %s %s",
					request.method,
					request.path,
				)
				return deleteAPIResponse{statusCode: http.StatusNotFound}
			}
		},
	)
	if err == nil {
		t.Fatal("Expected partial restart-series deletion to fail")
	}
	for _, want := range []string{
		"could not be fully deleted",
		"Deleted runs: #7, #7.2",
		"Failed runs: #7.1, #7.3",
		"First error: first sibling deletion failed",
		"--include-all-restarts",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Expected error to contain %q, got %q", want, err)
		}
	}
	if output != "" {
		t.Errorf("Got unexpected success output: %q", output)
	}
	if len(requests) != 6 {
		t.Fatalf("Got %d API requests, want 6", len(requests))
	}
	for _, path := range []string{
		"/api/workflows/run-7/status",
		"/api/workflows/run-7-1/status",
		"/api/workflows/run-7-2/status",
		"/api/workflows/run-7-3/status",
	} {
		if !hasDeleteRequest(requests, path) {
			t.Errorf("Missing deletion request for %s", path)
		}
	}
}

func TestDeleteRestartSeriesStopsAfterSelectedFailure(t *testing.T) {
	output, requests, err := executeDeleteCommand(
		t,
		[]string{"-w", "analysis.7", "--include-all-restarts"},
		func(request deleteAPIRequest) deleteAPIResponse {
			switch request.method + " " + request.path {
			case "GET /api/workflows/analysis.7/status":
				return deleteStatusResponse("run-7", "analysis.7", "finished")
			case "GET /api/workflows":
				return deleteListResponse(
					deleteRun("run-7", "analysis.7", "finished"),
					deleteRun("run-7-1", "analysis.7.1", "failed"),
					deleteRun("run-7-2", "analysis.7.2", "stopped"),
				)
			case "PUT /api/workflows/run-7/status":
				return deleteFailureResponse("selected deletion failed")
			default:
				t.Errorf(
					"Unexpected request after selected-run failure: %s %s",
					request.method,
					request.path,
				)
				return deleteAPIResponse{statusCode: http.StatusNotFound}
			}
		},
	)
	if err == nil {
		t.Fatal("Expected selected-run deletion to fail")
	}
	for _, want := range []string{
		"Deleted runs: none",
		"Failed runs: #7",
		"First error: selected deletion failed",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Expected error to contain %q, got %q", want, err)
		}
	}
	if output != "" {
		t.Errorf("Got unexpected success output: %q", output)
	}
	if len(requests) != 3 {
		t.Fatalf("Got %d API requests, want 3", len(requests))
	}
	if hasDeleteRequest(requests, "/api/workflows/run-7-1/status") ||
		hasDeleteRequest(requests, "/api/workflows/run-7-2/status") {
		t.Errorf(
			"Sibling deletion continued after selected failure: %+v",
			requests,
		)
	}
}

func TestActiveRestartLabels(t *testing.T) {
	statuses := []string{
		"created",
		"pending",
		"queued",
		"running",
		"finished",
		"failed",
		"stopped",
	}
	runs := make([]*operations.GetWorkflowsOKBodyItemsItems0, 0, len(statuses))
	for i, status := range statuses {
		runs = append(runs, &operations.GetWorkflowsOKBodyItemsItems0{
			Name:   fmt.Sprintf("analysis.7.%d", i+1),
			Status: status,
		})
	}

	got := activeRestartLabels(runs)
	want := []string{"#7.1", "#7.2", "#7.3", "#7.4"}
	if !slices.Equal(got, want) {
		t.Errorf("activeRestartLabels() = %v, want %v", got, want)
	}
}

func hasDeleteRequest(requests []deleteAPIRequest, path string) bool {
	for _, request := range requests {
		if request.method == http.MethodPut && request.path == path {
			return true
		}
	}
	return false
}
