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

// NewMoveFilesParams creates a new MoveFilesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewMoveFilesParams() *MoveFilesParams {
	return &MoveFilesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewMoveFilesParamsWithTimeout creates a new MoveFilesParams object
// with the ability to set a timeout on a request.
func NewMoveFilesParamsWithTimeout(timeout time.Duration) *MoveFilesParams {
	return &MoveFilesParams{
		timeout: timeout,
	}
}

// NewMoveFilesParamsWithContext creates a new MoveFilesParams object
// with the ability to set a context for a request.
func NewMoveFilesParamsWithContext(ctx context.Context) *MoveFilesParams {
	return &MoveFilesParams{
		Context: ctx,
	}
}

// NewMoveFilesParamsWithHTTPClient creates a new MoveFilesParams object
// with the ability to set a custom HTTPClient for a request.
func NewMoveFilesParamsWithHTTPClient(client *http.Client) *MoveFilesParams {
	return &MoveFilesParams{
		HTTPClient: client,
	}
}

/*
MoveFilesParams contains all the parameters to send to the API endpoint

	for the move files operation.

	Typically these are written to a http.Request.
*/
type MoveFilesParams struct {

	/* AccessToken.

	   The API access_token of workflow owner.
	*/
	AccessToken *string

	/* Source.

	   Required. Source file(s).
	*/
	Source string

	/* Target.

	   Required. Target file(s).
	*/
	Target string

	/* WorkflowIDOrName.

	   Required. Analysis UUID or name.
	*/
	WorkflowIDOrName string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the move files params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *MoveFilesParams) WithDefaults() *MoveFilesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the move files params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *MoveFilesParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the move files params
func (o *MoveFilesParams) WithTimeout(timeout time.Duration) *MoveFilesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the move files params
func (o *MoveFilesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the move files params
func (o *MoveFilesParams) WithContext(ctx context.Context) *MoveFilesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the move files params
func (o *MoveFilesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the move files params
func (o *MoveFilesParams) WithHTTPClient(client *http.Client) *MoveFilesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the move files params
func (o *MoveFilesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithAccessToken adds the accessToken to the move files params
func (o *MoveFilesParams) WithAccessToken(accessToken *string) *MoveFilesParams {
	o.SetAccessToken(accessToken)
	return o
}

// SetAccessToken adds the accessToken to the move files params
func (o *MoveFilesParams) SetAccessToken(accessToken *string) {
	o.AccessToken = accessToken
}

// WithSource adds the source to the move files params
func (o *MoveFilesParams) WithSource(source string) *MoveFilesParams {
	o.SetSource(source)
	return o
}

// SetSource adds the source to the move files params
func (o *MoveFilesParams) SetSource(source string) {
	o.Source = source
}

// WithTarget adds the target to the move files params
func (o *MoveFilesParams) WithTarget(target string) *MoveFilesParams {
	o.SetTarget(target)
	return o
}

// SetTarget adds the target to the move files params
func (o *MoveFilesParams) SetTarget(target string) {
	o.Target = target
}

// WithWorkflowIDOrName adds the workflowIDOrName to the move files params
func (o *MoveFilesParams) WithWorkflowIDOrName(workflowIDOrName string) *MoveFilesParams {
	o.SetWorkflowIDOrName(workflowIDOrName)
	return o
}

// SetWorkflowIDOrName adds the workflowIdOrName to the move files params
func (o *MoveFilesParams) SetWorkflowIDOrName(workflowIDOrName string) {
	o.WorkflowIDOrName = workflowIDOrName
}

// WriteToRequest writes these params to a swagger request
func (o *MoveFilesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	// query param source
	qrSource := o.Source
	qSource := qrSource
	if qSource != "" {

		if err := r.SetQueryParam("source", qSource); err != nil {
			return err
		}
	}

	// query param target
	qrTarget := o.Target
	qTarget := qrTarget
	if qTarget != "" {

		if err := r.SetQueryParam("target", qTarget); err != nil {
			return err
		}
	}

	// path param workflow_id_or_name
	if err := r.SetPathParam("workflow_id_or_name", o.WorkflowIDOrName); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
