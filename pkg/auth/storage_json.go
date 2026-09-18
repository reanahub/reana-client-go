/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package auth

import (
	"encoding/json"
	"reflect"
	"strings"
)

// TLSSettings stores explicit certificate policy; omitted verification is enabled.
type TLSSettings struct {
	Verify *bool `json:"verify,omitempty"`
	extra  map[string]json.RawMessage
}

// Preserve fields introduced by other versions of either client on every rewrite.
func decodeFields(data []byte, value any) (map[string]json.RawMessage, error) {
	if err := json.Unmarshal(data, value); err != nil {
		return nil, err
	}
	var extra map[string]json.RawMessage
	if err := json.Unmarshal(data, &extra); err != nil {
		return nil, err
	}
	typ := reflect.TypeOf(value).Elem()
	for i := 0; i < typ.NumField(); i++ {
		name := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		delete(extra, name)
	}
	return extra, nil
}

func encodeFields(value any, extra map[string]json.RawMessage) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	fields := make(map[string]json.RawMessage)
	if err = json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	for key, value := range extra {
		if _, exists := fields[key]; !exists {
			fields[key] = value
		}
	}
	return json.Marshal(fields)
}

type plainCredentials Credentials
type plainConfig credentialConfig
type plainTLS TLSSettings

// UnmarshalJSON remembers additional per-server fields.
func (c *Credentials) UnmarshalJSON(data []byte) (err error) {
	c.extra, err = decodeFields(data, (*plainCredentials)(c))
	return
}

// MarshalJSON retains additional per-server fields.
func (c Credentials) MarshalJSON() ([]byte, error) { return encodeFields(plainCredentials(c), c.extra) }
func (c *credentialConfig) UnmarshalJSON(data []byte) (err error) {
	c.extra, err = decodeFields(data, (*plainConfig)(c))
	return
}

func (c credentialConfig) MarshalJSON() ([]byte, error) {
	var active *string
	if c.ActiveServer != "" {
		active = &c.ActiveServer
	}
	value := struct {
		plainConfig
		ActiveServer *string `json:"active_server"`
	}{plainConfig(c), active}
	return encodeFields(value, c.extra)
}

// UnmarshalJSON remembers additional TLS settings.
func (c *TLSSettings) UnmarshalJSON(data []byte) (err error) {
	c.extra, err = decodeFields(data, (*plainTLS)(c))
	return
}

// MarshalJSON retains additional TLS settings.
func (c TLSSettings) MarshalJSON() ([]byte, error) { return encodeFields(plainTLS(c), c.extra) }

func preserveSettings(entry, previous Credentials) Credentials {
	entry.extra = previous.extra
	if entry.TLS == nil {
		entry.TLS = previous.TLS
	} else if previous.TLS != nil {
		entry.TLS.extra = previous.TLS.extra
	}
	return entry
}
