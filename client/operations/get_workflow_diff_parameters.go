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
	"github.com/go-openapi/swag"
)

// NewGetWorkflowDiffParams creates a new GetWorkflowDiffParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGetWorkflowDiffParams() *GetWorkflowDiffParams {
	return &GetWorkflowDiffParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGetWorkflowDiffParamsWithTimeout creates a new GetWorkflowDiffParams object
// with the ability to set a timeout on a request.
func NewGetWorkflowDiffParamsWithTimeout(timeout time.Duration) *GetWorkflowDiffParams {
	return &GetWorkflowDiffParams{
		timeout: timeout,
	}
}

// NewGetWorkflowDiffParamsWithContext creates a new GetWorkflowDiffParams object
// with the ability to set a context for a request.
func NewGetWorkflowDiffParamsWithContext(ctx context.Context) *GetWorkflowDiffParams {
	return &GetWorkflowDiffParams{
		Context: ctx,
	}
}

// NewGetWorkflowDiffParamsWithHTTPClient creates a new GetWorkflowDiffParams object
// with the ability to set a custom HTTPClient for a request.
func NewGetWorkflowDiffParamsWithHTTPClient(client *http.Client) *GetWorkflowDiffParams {
	return &GetWorkflowDiffParams{
		HTTPClient: client,
	}
}

/* GetWorkflowDiffParams contains all the parameters to send to the API endpoint
   for the get workflow diff operation.

   Typically these are written to a http.Request.
*/
type GetWorkflowDiffParams struct {

	/* AccessToken.

	   The API access_token of workflow owner.
	*/
	AccessToken *string

	/* Brief.

	   Optional flag. If set, file contents are examined.
	*/
	Brief *bool

	/* ContextLines.

	   Optional parameter. Sets number of context lines for workspace diff output.

	   Default: "5"
	*/
	ContextLines *string

	/* WorkflowIDOrNamea.

	   Required. Analysis UUID or name of the first workflow.
	*/
	WorkflowIDOrNamea string

	/* WorkflowIDOrNameb.

	   Required. Analysis UUID or name of the second workflow.
	*/
	WorkflowIDOrNameb string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the get workflow diff params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetWorkflowDiffParams) WithDefaults() *GetWorkflowDiffParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the get workflow diff params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetWorkflowDiffParams) SetDefaults() {
	var (
		briefDefault = bool(false)

		contextLinesDefault = string("5")
	)

	val := GetWorkflowDiffParams{
		Brief:        &briefDefault,
		ContextLines: &contextLinesDefault,
	}

	val.timeout = o.timeout
	val.Context = o.Context
	val.HTTPClient = o.HTTPClient
	*o = val
}

// WithTimeout adds the timeout to the get workflow diff params
func (o *GetWorkflowDiffParams) WithTimeout(timeout time.Duration) *GetWorkflowDiffParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get workflow diff params
func (o *GetWorkflowDiffParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get workflow diff params
func (o *GetWorkflowDiffParams) WithContext(ctx context.Context) *GetWorkflowDiffParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get workflow diff params
func (o *GetWorkflowDiffParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get workflow diff params
func (o *GetWorkflowDiffParams) WithHTTPClient(client *http.Client) *GetWorkflowDiffParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get workflow diff params
func (o *GetWorkflowDiffParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithAccessToken adds the accessToken to the get workflow diff params
func (o *GetWorkflowDiffParams) WithAccessToken(accessToken *string) *GetWorkflowDiffParams {
	o.SetAccessToken(accessToken)
	return o
}

// SetAccessToken adds the accessToken to the get workflow diff params
func (o *GetWorkflowDiffParams) SetAccessToken(accessToken *string) {
	o.AccessToken = accessToken
}

// WithBrief adds the brief to the get workflow diff params
func (o *GetWorkflowDiffParams) WithBrief(brief *bool) *GetWorkflowDiffParams {
	o.SetBrief(brief)
	return o
}

// SetBrief adds the brief to the get workflow diff params
func (o *GetWorkflowDiffParams) SetBrief(brief *bool) {
	o.Brief = brief
}

// WithContextLines adds the contextLines to the get workflow diff params
func (o *GetWorkflowDiffParams) WithContextLines(contextLines *string) *GetWorkflowDiffParams {
	o.SetContextLines(contextLines)
	return o
}

// SetContextLines adds the contextLines to the get workflow diff params
func (o *GetWorkflowDiffParams) SetContextLines(contextLines *string) {
	o.ContextLines = contextLines
}

// WithWorkflowIDOrNamea adds the workflowIDOrNamea to the get workflow diff params
func (o *GetWorkflowDiffParams) WithWorkflowIDOrNamea(workflowIDOrNamea string) *GetWorkflowDiffParams {
	o.SetWorkflowIDOrNamea(workflowIDOrNamea)
	return o
}

// SetWorkflowIDOrNamea adds the workflowIdOrNameA to the get workflow diff params
func (o *GetWorkflowDiffParams) SetWorkflowIDOrNamea(workflowIDOrNamea string) {
	o.WorkflowIDOrNamea = workflowIDOrNamea
}

// WithWorkflowIDOrNameb adds the workflowIDOrNameb to the get workflow diff params
func (o *GetWorkflowDiffParams) WithWorkflowIDOrNameb(workflowIDOrNameb string) *GetWorkflowDiffParams {
	o.SetWorkflowIDOrNameb(workflowIDOrNameb)
	return o
}

// SetWorkflowIDOrNameb adds the workflowIdOrNameB to the get workflow diff params
func (o *GetWorkflowDiffParams) SetWorkflowIDOrNameb(workflowIDOrNameb string) {
	o.WorkflowIDOrNameb = workflowIDOrNameb
}

// WriteToRequest writes these params to a swagger request
func (o *GetWorkflowDiffParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	if o.Brief != nil {

		// query param brief
		var qrBrief bool

		if o.Brief != nil {
			qrBrief = *o.Brief
		}
		qBrief := swag.FormatBool(qrBrief)
		if qBrief != "" {

			if err := r.SetQueryParam("brief", qBrief); err != nil {
				return err
			}
		}
	}

	if o.ContextLines != nil {

		// query param context_lines
		var qrContextLines string

		if o.ContextLines != nil {
			qrContextLines = *o.ContextLines
		}
		qContextLines := qrContextLines
		if qContextLines != "" {

			if err := r.SetQueryParam("context_lines", qContextLines); err != nil {
				return err
			}
		}
	}

	// path param workflow_id_or_name_a
	if err := r.SetPathParam("workflow_id_or_name_a", o.WorkflowIDOrNamea); err != nil {
		return err
	}

	// path param workflow_id_or_name_b
	if err := r.SetPathParam("workflow_id_or_name_b", o.WorkflowIDOrNameb); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}