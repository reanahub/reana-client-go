/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"reanahub/reana-client-go/validation"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Check connection to REANA server.",
	Long: `
Check connection to REANA server.

The ` + "``list``" + ` command allows to test connection to REANA server.

Examples:

  $ reana-client list
	`,
	Run: func(cmd *cobra.Command, args []string) {
		token, _ := cmd.Flags().GetString("access-token")
		serverURL := os.Getenv("REANA_SERVER_URL")
		validation.ValidateAccessToken(token)
		validation.ValidateServerURL(serverURL)
		list(token, serverURL)
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringP("access-token", "t", os.Getenv("REANA_ACCESS_TOKEN"), "Access token of the current user.")
}

func list(accessToken string, serverURL string) {
		// disable certificate security checks
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}

		// make API query
		resp, err := http.Get(
			serverURL + "/api/workflows?type=workflow&access_token=" + accessToken,
		)
		if err != nil {
			fmt.Println(err)
			os.Exit(3)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			os.Exit(4)
		}

		// define response structure
		type rList struct {
			HasNext bool `json:"has_next"`
			HasPrev bool `json:"has_prev"`
			Items   []struct {
				Created  string `json:"created"`
				ID       string `json:"id"`
				Name     string `json:"name"`
				Progress struct {
					RunFinishedAt string `json:"run_finished_at"`
					RunStartedAt  string `json:"run_started_at"`
				} `json:"progress"`
				Size struct {
					HumanReadable string `json:"human_readable"`
					Raw           int    `json:"raw"`
				} `json:"size"`
				Status string `json:"status"`
				User   string `json:"user"`
			} `json:"items"`
			Page             int  `json:"page"`
			Total            int  `json:"total"`
			UserHasWorkflows bool `json:"user_has_workflows"`
		}

		// parse response
		p := rList{}
		err = json.Unmarshal(body, &p)
		if err != nil {
			panic(err)
		}

		// format output
		fmt.Printf(
			"%-38s %-12s %-21s %-21s %-21s %-8s\n",
			"NAME",
			"RUN_NUMBER",
			"CREATED",
			"STARTED",
			"ENDED",
			"STATUS",
		)
		for _, workflow := range p.Items {
			workflowNameAndRunnumber := strings.SplitN(workflow.Name, ".", 2)
			fmt.Printf(
				"%-38s %-12s %-21s %-21s %-21s %-8s\n",
				workflowNameAndRunnumber[0],
				workflowNameAndRunnumber[1],
				workflow.Created,
				workflow.Progress.RunStartedAt,
				workflow.Progress.RunFinishedAt,
				workflow.Status,
			)
		}
}
