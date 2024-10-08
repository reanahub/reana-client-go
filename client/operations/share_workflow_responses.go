// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// ShareWorkflowReader is a Reader for the ShareWorkflow structure.
type ShareWorkflowReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ShareWorkflowReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewShareWorkflowOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 400:
		result := NewShareWorkflowBadRequest()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 401:
		result := NewShareWorkflowUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewShareWorkflowForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 404:
		result := NewShareWorkflowNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 409:
		result := NewShareWorkflowConflict()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewShareWorkflowInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("[POST /api/workflows/{workflow_id_or_name}/share] share_workflow", response, response.Code())
	}
}

// NewShareWorkflowOK creates a ShareWorkflowOK with default headers values
func NewShareWorkflowOK() *ShareWorkflowOK {
	return &ShareWorkflowOK{}
}

/*
ShareWorkflowOK describes a response with status code 200, with default header values.

Request succeeded. The workflow has been shared with the user.
*/
type ShareWorkflowOK struct {
	Payload *ShareWorkflowOKBody
}

// IsSuccess returns true when this share workflow o k response has a 2xx status code
func (o *ShareWorkflowOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this share workflow o k response has a 3xx status code
func (o *ShareWorkflowOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this share workflow o k response has a 4xx status code
func (o *ShareWorkflowOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this share workflow o k response has a 5xx status code
func (o *ShareWorkflowOK) IsServerError() bool {
	return false
}

// IsCode returns true when this share workflow o k response a status code equal to that given
func (o *ShareWorkflowOK) IsCode(code int) bool {
	return code == 200
}

// Code gets the status code for the share workflow o k response
func (o *ShareWorkflowOK) Code() int {
	return 200
}

func (o *ShareWorkflowOK) Error() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /api/workflows/{workflow_id_or_name}/share][%d] shareWorkflowOK %s", 200, payload)
}

func (o *ShareWorkflowOK) String() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /api/workflows/{workflow_id_or_name}/share][%d] shareWorkflowOK %s", 200, payload)
}

func (o *ShareWorkflowOK) GetPayload() *ShareWorkflowOKBody {
	return o.Payload
}

func (o *ShareWorkflowOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ShareWorkflowOKBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewShareWorkflowBadRequest creates a ShareWorkflowBadRequest with default headers values
func NewShareWorkflowBadRequest() *ShareWorkflowBadRequest {
	return &ShareWorkflowBadRequest{}
}

/*
ShareWorkflowBadRequest describes a response with status code 400, with default header values.

Request failed. The incoming data seems malformed.
*/
type ShareWorkflowBadRequest struct {
	Payload *ShareWorkflowBadRequestBody
}

// IsSuccess returns true when this share workflow bad request response has a 2xx status code
func (o *ShareWorkflowBadRequest) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this share workflow bad request response has a 3xx status code
func (o *ShareWorkflowBadRequest) IsRedirect() bool {
	return false
}

// IsClientError returns true when this share workflow bad request response has a 4xx status code
func (o *ShareWorkflowBadRequest) IsClientError() bool {
	return true
}

// IsServerError returns true when this share workflow bad request response has a 5xx status code
func (o *ShareWorkflowBadRequest) IsServerError() bool {
	return false
}

// IsCode returns true when this share workflow bad request response a status code equal to that given
func (o *ShareWorkflowBadRequest) IsCode(code int) bool {
	return code == 400
}

// Code gets the status code for the share workflow bad request response
func (o *ShareWorkflowBadRequest) Code() int {
	return 400
}

func (o *ShareWorkflowBadRequest) Error() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /api/workflows/{workflow_id_or_name}/share][%d] shareWorkflowBadRequest %s", 400, payload)
}

func (o *ShareWorkflowBadRequest) String() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /api/workflows/{workflow_id_or_name}/share][%d] shareWorkflowBadRequest %s", 400, payload)
}

func (o *ShareWorkflowBadRequest) GetPayload() *ShareWorkflowBadRequestBody {
	return o.Payload
}

func (o *ShareWorkflowBadRequest) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ShareWorkflowBadRequestBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewShareWorkflowUnauthorized creates a ShareWorkflowUnauthorized with default headers values
func NewShareWorkflowUnauthorized() *ShareWorkflowUnauthorized {
	return &ShareWorkflowUnauthorized{}
}

/*
ShareWorkflowUnauthorized describes a response with status code 401, with default header values.

Request failed. User not signed in.
*/
type ShareWorkflowUnauthorized struct {
	Payload *ShareWorkflowUnauthorizedBody
}

// IsSuccess returns true when this share workflow unauthorized response has a 2xx status code
func (o *ShareWorkflowUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this share workflow unauthorized response has a 3xx status code
func (o *ShareWorkflowUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this share workflow unauthorized response has a 4xx status code
func (o *ShareWorkflowUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this share workflow unauthorized response has a 5xx status code
func (o *ShareWorkflowUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this share workflow unauthorized response a status code equal to that given
func (o *ShareWorkflowUnauthorized) IsCode(code int) bool {
	return code == 401
}

// Code gets the status code for the share workflow unauthorized response
func (o *ShareWorkflowUnauthorized) Code() int {
	return 401
}

func (o *ShareWorkflowUnauthorized) Error() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /api/workflows/{workflow_id_or_name}/share][%d] shareWorkflowUnauthorized %s", 401, payload)
}

func (o *ShareWorkflowUnauthorized) String() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /api/workflows/{workflow_id_or_name}/share][%d] shareWorkflowUnauthorized %s", 401, payload)
}

func (o *ShareWorkflowUnauthorized) GetPayload() *ShareWorkflowUnauthorizedBody {
	return o.Payload
}

func (o *ShareWorkflowUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ShareWorkflowUnauthorizedBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewShareWorkflowForbidden creates a ShareWorkflowForbidden with default headers values
func NewShareWorkflowForbidden() *ShareWorkflowForbidden {
	return &ShareWorkflowForbidden{}
}

/*
ShareWorkflowForbidden describes a response with status code 403, with default header values.

Request failed. Credentials are invalid or revoked.
*/
type ShareWorkflowForbidden struct {
	Payload *ShareWorkflowForbiddenBody
}

// IsSuccess returns true when this share workflow forbidden response has a 2xx status code
func (o *ShareWorkflowForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this share workflow forbidden response has a 3xx status code
func (o *ShareWorkflowForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this share workflow forbidden response has a 4xx status code
func (o *ShareWorkflowForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this share workflow forbidden response has a 5xx status code
func (o *ShareWorkflowForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this share workflow forbidden response a status code equal to that given
func (o *ShareWorkflowForbidden) IsCode(code int) bool {
	return code == 403
}

// Code gets the status code for the share workflow forbidden response
func (o *ShareWorkflowForbidden) Code() int {
	return 403
}

func (o *ShareWorkflowForbidden) Error() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /api/workflows/{workflow_id_or_name}/share][%d] shareWorkflowForbidden %s", 403, payload)
}

func (o *ShareWorkflowForbidden) String() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /api/workflows/{workflow_id_or_name}/share][%d] shareWorkflowForbidden %s", 403, payload)
}

func (o *ShareWorkflowForbidden) GetPayload() *ShareWorkflowForbiddenBody {
	return o.Payload
}

func (o *ShareWorkflowForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ShareWorkflowForbiddenBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewShareWorkflowNotFound creates a ShareWorkflowNotFound with default headers values
func NewShareWorkflowNotFound() *ShareWorkflowNotFound {
	return &ShareWorkflowNotFound{}
}

/*
ShareWorkflowNotFound describes a response with status code 404, with default header values.

Request failed. Workflow does not exist or user does not exist.
*/
type ShareWorkflowNotFound struct {
	Payload *ShareWorkflowNotFoundBody
}

// IsSuccess returns true when this share workflow not found response has a 2xx status code
func (o *ShareWorkflowNotFound) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this share workflow not found response has a 3xx status code
func (o *ShareWorkflowNotFound) IsRedirect() bool {
	return false
}

// IsClientError returns true when this share workflow not found response has a 4xx status code
func (o *ShareWorkflowNotFound) IsClientError() bool {
	return true
}

// IsServerError returns true when this share workflow not found response has a 5xx status code
func (o *ShareWorkflowNotFound) IsServerError() bool {
	return false
}

// IsCode returns true when this share workflow not found response a status code equal to that given
func (o *ShareWorkflowNotFound) IsCode(code int) bool {
	return code == 404
}

// Code gets the status code for the share workflow not found response
func (o *ShareWorkflowNotFound) Code() int {
	return 404
}

func (o *ShareWorkflowNotFound) Error() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /api/workflows/{workflow_id_or_name}/share][%d] shareWorkflowNotFound %s", 404, payload)
}

func (o *ShareWorkflowNotFound) String() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /api/workflows/{workflow_id_or_name}/share][%d] shareWorkflowNotFound %s", 404, payload)
}

func (o *ShareWorkflowNotFound) GetPayload() *ShareWorkflowNotFoundBody {
	return o.Payload
}

func (o *ShareWorkflowNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ShareWorkflowNotFoundBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewShareWorkflowConflict creates a ShareWorkflowConflict with default headers values
func NewShareWorkflowConflict() *ShareWorkflowConflict {
	return &ShareWorkflowConflict{}
}

/*
ShareWorkflowConflict describes a response with status code 409, with default header values.

Request failed. The workflow is already shared with the user.
*/
type ShareWorkflowConflict struct {
	Payload *ShareWorkflowConflictBody
}

// IsSuccess returns true when this share workflow conflict response has a 2xx status code
func (o *ShareWorkflowConflict) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this share workflow conflict response has a 3xx status code
func (o *ShareWorkflowConflict) IsRedirect() bool {
	return false
}

// IsClientError returns true when this share workflow conflict response has a 4xx status code
func (o *ShareWorkflowConflict) IsClientError() bool {
	return true
}

// IsServerError returns true when this share workflow conflict response has a 5xx status code
func (o *ShareWorkflowConflict) IsServerError() bool {
	return false
}

// IsCode returns true when this share workflow conflict response a status code equal to that given
func (o *ShareWorkflowConflict) IsCode(code int) bool {
	return code == 409
}

// Code gets the status code for the share workflow conflict response
func (o *ShareWorkflowConflict) Code() int {
	return 409
}

func (o *ShareWorkflowConflict) Error() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /api/workflows/{workflow_id_or_name}/share][%d] shareWorkflowConflict %s", 409, payload)
}

func (o *ShareWorkflowConflict) String() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /api/workflows/{workflow_id_or_name}/share][%d] shareWorkflowConflict %s", 409, payload)
}

func (o *ShareWorkflowConflict) GetPayload() *ShareWorkflowConflictBody {
	return o.Payload
}

func (o *ShareWorkflowConflict) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ShareWorkflowConflictBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewShareWorkflowInternalServerError creates a ShareWorkflowInternalServerError with default headers values
func NewShareWorkflowInternalServerError() *ShareWorkflowInternalServerError {
	return &ShareWorkflowInternalServerError{}
}

/*
ShareWorkflowInternalServerError describes a response with status code 500, with default header values.

Request failed. Internal controller error.
*/
type ShareWorkflowInternalServerError struct {
	Payload *ShareWorkflowInternalServerErrorBody
}

// IsSuccess returns true when this share workflow internal server error response has a 2xx status code
func (o *ShareWorkflowInternalServerError) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this share workflow internal server error response has a 3xx status code
func (o *ShareWorkflowInternalServerError) IsRedirect() bool {
	return false
}

// IsClientError returns true when this share workflow internal server error response has a 4xx status code
func (o *ShareWorkflowInternalServerError) IsClientError() bool {
	return false
}

// IsServerError returns true when this share workflow internal server error response has a 5xx status code
func (o *ShareWorkflowInternalServerError) IsServerError() bool {
	return true
}

// IsCode returns true when this share workflow internal server error response a status code equal to that given
func (o *ShareWorkflowInternalServerError) IsCode(code int) bool {
	return code == 500
}

// Code gets the status code for the share workflow internal server error response
func (o *ShareWorkflowInternalServerError) Code() int {
	return 500
}

func (o *ShareWorkflowInternalServerError) Error() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /api/workflows/{workflow_id_or_name}/share][%d] shareWorkflowInternalServerError %s", 500, payload)
}

func (o *ShareWorkflowInternalServerError) String() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /api/workflows/{workflow_id_or_name}/share][%d] shareWorkflowInternalServerError %s", 500, payload)
}

func (o *ShareWorkflowInternalServerError) GetPayload() *ShareWorkflowInternalServerErrorBody {
	return o.Payload
}

func (o *ShareWorkflowInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(ShareWorkflowInternalServerErrorBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

/*
ShareWorkflowBadRequestBody share workflow bad request body
swagger:model ShareWorkflowBadRequestBody
*/
type ShareWorkflowBadRequestBody struct {

	// message
	Message string `json:"message,omitempty"`
}

// Validate validates this share workflow bad request body
func (o *ShareWorkflowBadRequestBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this share workflow bad request body based on context it is used
func (o *ShareWorkflowBadRequestBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ShareWorkflowBadRequestBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ShareWorkflowBadRequestBody) UnmarshalBinary(b []byte) error {
	var res ShareWorkflowBadRequestBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
ShareWorkflowBody share workflow body
swagger:model ShareWorkflowBody
*/
type ShareWorkflowBody struct {

	// Optional. Message to include when sharing the workflow.
	Message string `json:"message,omitempty"`

	// User to share the workflow with.
	// Required: true
	UserEmailToShareWith *string `json:"user_email_to_share_with"`

	// Optional. Date when access to the workflow will expire (format YYYY-MM-DD).
	ValidUntil string `json:"valid_until,omitempty"`
}

// Validate validates this share workflow body
func (o *ShareWorkflowBody) Validate(formats strfmt.Registry) error {
	var res []error

	if err := o.validateUserEmailToShareWith(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *ShareWorkflowBody) validateUserEmailToShareWith(formats strfmt.Registry) error {

	if err := validate.Required("share_details"+"."+"user_email_to_share_with", "body", o.UserEmailToShareWith); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this share workflow body based on context it is used
func (o *ShareWorkflowBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ShareWorkflowBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ShareWorkflowBody) UnmarshalBinary(b []byte) error {
	var res ShareWorkflowBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
ShareWorkflowConflictBody share workflow conflict body
swagger:model ShareWorkflowConflictBody
*/
type ShareWorkflowConflictBody struct {

	// message
	Message string `json:"message,omitempty"`
}

// Validate validates this share workflow conflict body
func (o *ShareWorkflowConflictBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this share workflow conflict body based on context it is used
func (o *ShareWorkflowConflictBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ShareWorkflowConflictBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ShareWorkflowConflictBody) UnmarshalBinary(b []byte) error {
	var res ShareWorkflowConflictBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
ShareWorkflowForbiddenBody share workflow forbidden body
swagger:model ShareWorkflowForbiddenBody
*/
type ShareWorkflowForbiddenBody struct {

	// message
	Message string `json:"message,omitempty"`
}

// Validate validates this share workflow forbidden body
func (o *ShareWorkflowForbiddenBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this share workflow forbidden body based on context it is used
func (o *ShareWorkflowForbiddenBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ShareWorkflowForbiddenBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ShareWorkflowForbiddenBody) UnmarshalBinary(b []byte) error {
	var res ShareWorkflowForbiddenBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
ShareWorkflowInternalServerErrorBody share workflow internal server error body
swagger:model ShareWorkflowInternalServerErrorBody
*/
type ShareWorkflowInternalServerErrorBody struct {

	// message
	Message string `json:"message,omitempty"`
}

// Validate validates this share workflow internal server error body
func (o *ShareWorkflowInternalServerErrorBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this share workflow internal server error body based on context it is used
func (o *ShareWorkflowInternalServerErrorBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ShareWorkflowInternalServerErrorBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ShareWorkflowInternalServerErrorBody) UnmarshalBinary(b []byte) error {
	var res ShareWorkflowInternalServerErrorBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
ShareWorkflowNotFoundBody share workflow not found body
swagger:model ShareWorkflowNotFoundBody
*/
type ShareWorkflowNotFoundBody struct {

	// message
	Message string `json:"message,omitempty"`
}

// Validate validates this share workflow not found body
func (o *ShareWorkflowNotFoundBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this share workflow not found body based on context it is used
func (o *ShareWorkflowNotFoundBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ShareWorkflowNotFoundBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ShareWorkflowNotFoundBody) UnmarshalBinary(b []byte) error {
	var res ShareWorkflowNotFoundBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
ShareWorkflowOKBody share workflow o k body
swagger:model ShareWorkflowOKBody
*/
type ShareWorkflowOKBody struct {

	// message
	Message string `json:"message,omitempty"`

	// workflow id
	WorkflowID string `json:"workflow_id,omitempty"`

	// workflow name
	WorkflowName string `json:"workflow_name,omitempty"`
}

// Validate validates this share workflow o k body
func (o *ShareWorkflowOKBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this share workflow o k body based on context it is used
func (o *ShareWorkflowOKBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ShareWorkflowOKBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ShareWorkflowOKBody) UnmarshalBinary(b []byte) error {
	var res ShareWorkflowOKBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
ShareWorkflowUnauthorizedBody share workflow unauthorized body
swagger:model ShareWorkflowUnauthorizedBody
*/
type ShareWorkflowUnauthorizedBody struct {

	// message
	Message string `json:"message,omitempty"`
}

// Validate validates this share workflow unauthorized body
func (o *ShareWorkflowUnauthorizedBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this share workflow unauthorized body based on context it is used
func (o *ShareWorkflowUnauthorizedBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *ShareWorkflowUnauthorizedBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *ShareWorkflowUnauthorizedBody) UnmarshalBinary(b []byte) error {
	var res ShareWorkflowUnauthorizedBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}
