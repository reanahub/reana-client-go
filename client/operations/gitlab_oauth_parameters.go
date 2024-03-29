// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

// NewGitlabOauthParams creates a new GitlabOauthParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGitlabOauthParams() *GitlabOauthParams {
	return &GitlabOauthParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGitlabOauthParamsWithTimeout creates a new GitlabOauthParams object
// with the ability to set a timeout on a request.
func NewGitlabOauthParamsWithTimeout(timeout time.Duration) *GitlabOauthParams {
	return &GitlabOauthParams{
		timeout: timeout,
	}
}

// NewGitlabOauthParamsWithContext creates a new GitlabOauthParams object
// with the ability to set a context for a request.
func NewGitlabOauthParamsWithContext(ctx context.Context) *GitlabOauthParams {
	return &GitlabOauthParams{
		Context: ctx,
	}
}

// NewGitlabOauthParamsWithHTTPClient creates a new GitlabOauthParams object
// with the ability to set a custom HTTPClient for a request.
func NewGitlabOauthParamsWithHTTPClient(client *http.Client) *GitlabOauthParams {
	return &GitlabOauthParams{
		HTTPClient: client,
	}
}

/*
GitlabOauthParams contains all the parameters to send to the API endpoint

	for the gitlab oauth operation.

	Typically these are written to a http.Request.
*/
type GitlabOauthParams struct {
	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the gitlab oauth params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GitlabOauthParams) WithDefaults() *GitlabOauthParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the gitlab oauth params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GitlabOauthParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the gitlab oauth params
func (o *GitlabOauthParams) WithTimeout(timeout time.Duration) *GitlabOauthParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the gitlab oauth params
func (o *GitlabOauthParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the gitlab oauth params
func (o *GitlabOauthParams) WithContext(ctx context.Context) *GitlabOauthParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the gitlab oauth params
func (o *GitlabOauthParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the gitlab oauth params
func (o *GitlabOauthParams) WithHTTPClient(client *http.Client) *GitlabOauthParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the gitlab oauth params
func (o *GitlabOauthParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WriteToRequest writes these params to a swagger request
func (o *GitlabOauthParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
