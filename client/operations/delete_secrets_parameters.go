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

// NewDeleteSecretsParams creates a new DeleteSecretsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewDeleteSecretsParams() *DeleteSecretsParams {
	return &DeleteSecretsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewDeleteSecretsParamsWithTimeout creates a new DeleteSecretsParams object
// with the ability to set a timeout on a request.
func NewDeleteSecretsParamsWithTimeout(timeout time.Duration) *DeleteSecretsParams {
	return &DeleteSecretsParams{
		timeout: timeout,
	}
}

// NewDeleteSecretsParamsWithContext creates a new DeleteSecretsParams object
// with the ability to set a context for a request.
func NewDeleteSecretsParamsWithContext(ctx context.Context) *DeleteSecretsParams {
	return &DeleteSecretsParams{
		Context: ctx,
	}
}

// NewDeleteSecretsParamsWithHTTPClient creates a new DeleteSecretsParams object
// with the ability to set a custom HTTPClient for a request.
func NewDeleteSecretsParamsWithHTTPClient(client *http.Client) *DeleteSecretsParams {
	return &DeleteSecretsParams{
		HTTPClient: client,
	}
}

/*
DeleteSecretsParams contains all the parameters to send to the API endpoint

	for the delete secrets operation.

	Typically these are written to a http.Request.
*/
type DeleteSecretsParams struct {

	/* AccessToken.

	   API key of the admin.
	*/
	AccessToken *string

	/* Secrets.

	   Optional. List of secrets to be deleted.
	*/
	Secrets []string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the delete secrets params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteSecretsParams) WithDefaults() *DeleteSecretsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the delete secrets params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteSecretsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the delete secrets params
func (o *DeleteSecretsParams) WithTimeout(timeout time.Duration) *DeleteSecretsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the delete secrets params
func (o *DeleteSecretsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the delete secrets params
func (o *DeleteSecretsParams) WithContext(ctx context.Context) *DeleteSecretsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the delete secrets params
func (o *DeleteSecretsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the delete secrets params
func (o *DeleteSecretsParams) WithHTTPClient(client *http.Client) *DeleteSecretsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the delete secrets params
func (o *DeleteSecretsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithAccessToken adds the accessToken to the delete secrets params
func (o *DeleteSecretsParams) WithAccessToken(accessToken *string) *DeleteSecretsParams {
	o.SetAccessToken(accessToken)
	return o
}

// SetAccessToken adds the accessToken to the delete secrets params
func (o *DeleteSecretsParams) SetAccessToken(accessToken *string) {
	o.AccessToken = accessToken
}

// WithSecrets adds the secrets to the delete secrets params
func (o *DeleteSecretsParams) WithSecrets(secrets []string) *DeleteSecretsParams {
	o.SetSecrets(secrets)
	return o
}

// SetSecrets adds the secrets to the delete secrets params
func (o *DeleteSecretsParams) SetSecrets(secrets []string) {
	o.Secrets = secrets
}

// WriteToRequest writes these params to a swagger request
func (o *DeleteSecretsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.AccessToken != nil {

		// query param access_token
		var qrAccessToken string

		if o.AccessToken != nil {
			qrAccessToken = *o.AccessToken
		}
		qAccessToken := qrAccessToken
		if qAccessToken != "" {

			if err := r.SetQueryParam("access_token", qAccessToken); err != nil {
				return err
			}
		}
	}
	if o.Secrets != nil {
		if err := r.SetBodyParam(o.Secrets); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
