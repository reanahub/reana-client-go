/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package utils

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

func ExecuteCommand(root *cobra.Command, args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	err = root.Execute()

	return buf.String(), err
}

func DisplayJsonOutput(output any) {
	byteArray, err := json.MarshalIndent(output, "", "  ")

	if err != nil {
		fmt.Println("Error: ", err)
	}

	fmt.Println(string(byteArray))
}

func NewRequest(token string, serverURL string, endpoint string) []byte {
	// disable certificate security checks
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	url := serverURL + endpoint + "?access_token=" + token
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	return respBytes
}
