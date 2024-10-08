// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// GetUsersSharedWithYouReader is a Reader for the GetUsersSharedWithYou structure.
type GetUsersSharedWithYouReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetUsersSharedWithYouReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetUsersSharedWithYouOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewGetUsersSharedWithYouUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewGetUsersSharedWithYouForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewGetUsersSharedWithYouInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("[GET /api/users/shared-with-you] get_users_shared_with_you", response, response.Code())
	}
}

// NewGetUsersSharedWithYouOK creates a GetUsersSharedWithYouOK with default headers values
func NewGetUsersSharedWithYouOK() *GetUsersSharedWithYouOK {
	return &GetUsersSharedWithYouOK{}
}

/*
GetUsersSharedWithYouOK describes a response with status code 200, with default header values.

Users that shared workflow(s) with the authenticated user.
*/
type GetUsersSharedWithYouOK struct {
	Payload *GetUsersSharedWithYouOKBody
}

// IsSuccess returns true when this get users shared with you o k response has a 2xx status code
func (o *GetUsersSharedWithYouOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this get users shared with you o k response has a 3xx status code
func (o *GetUsersSharedWithYouOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get users shared with you o k response has a 4xx status code
func (o *GetUsersSharedWithYouOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this get users shared with you o k response has a 5xx status code
func (o *GetUsersSharedWithYouOK) IsServerError() bool {
	return false
}

// IsCode returns true when this get users shared with you o k response a status code equal to that given
func (o *GetUsersSharedWithYouOK) IsCode(code int) bool {
	return code == 200
}

// Code gets the status code for the get users shared with you o k response
func (o *GetUsersSharedWithYouOK) Code() int {
	return 200
}

func (o *GetUsersSharedWithYouOK) Error() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[GET /api/users/shared-with-you][%d] getUsersSharedWithYouOK %s", 200, payload)
}

func (o *GetUsersSharedWithYouOK) String() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[GET /api/users/shared-with-you][%d] getUsersSharedWithYouOK %s", 200, payload)
}

func (o *GetUsersSharedWithYouOK) GetPayload() *GetUsersSharedWithYouOKBody {
	return o.Payload
}

func (o *GetUsersSharedWithYouOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(GetUsersSharedWithYouOKBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetUsersSharedWithYouUnauthorized creates a GetUsersSharedWithYouUnauthorized with default headers values
func NewGetUsersSharedWithYouUnauthorized() *GetUsersSharedWithYouUnauthorized {
	return &GetUsersSharedWithYouUnauthorized{}
}

/*
GetUsersSharedWithYouUnauthorized describes a response with status code 401, with default header values.

Error message indicating that the uses is not authenticated.
*/
type GetUsersSharedWithYouUnauthorized struct {
	Payload *GetUsersSharedWithYouUnauthorizedBody
}

// IsSuccess returns true when this get users shared with you unauthorized response has a 2xx status code
func (o *GetUsersSharedWithYouUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get users shared with you unauthorized response has a 3xx status code
func (o *GetUsersSharedWithYouUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get users shared with you unauthorized response has a 4xx status code
func (o *GetUsersSharedWithYouUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this get users shared with you unauthorized response has a 5xx status code
func (o *GetUsersSharedWithYouUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this get users shared with you unauthorized response a status code equal to that given
func (o *GetUsersSharedWithYouUnauthorized) IsCode(code int) bool {
	return code == 401
}

// Code gets the status code for the get users shared with you unauthorized response
func (o *GetUsersSharedWithYouUnauthorized) Code() int {
	return 401
}

func (o *GetUsersSharedWithYouUnauthorized) Error() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[GET /api/users/shared-with-you][%d] getUsersSharedWithYouUnauthorized %s", 401, payload)
}

func (o *GetUsersSharedWithYouUnauthorized) String() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[GET /api/users/shared-with-you][%d] getUsersSharedWithYouUnauthorized %s", 401, payload)
}

func (o *GetUsersSharedWithYouUnauthorized) GetPayload() *GetUsersSharedWithYouUnauthorizedBody {
	return o.Payload
}

func (o *GetUsersSharedWithYouUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(GetUsersSharedWithYouUnauthorizedBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetUsersSharedWithYouForbidden creates a GetUsersSharedWithYouForbidden with default headers values
func NewGetUsersSharedWithYouForbidden() *GetUsersSharedWithYouForbidden {
	return &GetUsersSharedWithYouForbidden{}
}

/*
GetUsersSharedWithYouForbidden describes a response with status code 403, with default header values.

Request failed. User token not valid.
*/
type GetUsersSharedWithYouForbidden struct {
	Payload *GetUsersSharedWithYouForbiddenBody
}

// IsSuccess returns true when this get users shared with you forbidden response has a 2xx status code
func (o *GetUsersSharedWithYouForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get users shared with you forbidden response has a 3xx status code
func (o *GetUsersSharedWithYouForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get users shared with you forbidden response has a 4xx status code
func (o *GetUsersSharedWithYouForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this get users shared with you forbidden response has a 5xx status code
func (o *GetUsersSharedWithYouForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this get users shared with you forbidden response a status code equal to that given
func (o *GetUsersSharedWithYouForbidden) IsCode(code int) bool {
	return code == 403
}

// Code gets the status code for the get users shared with you forbidden response
func (o *GetUsersSharedWithYouForbidden) Code() int {
	return 403
}

func (o *GetUsersSharedWithYouForbidden) Error() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[GET /api/users/shared-with-you][%d] getUsersSharedWithYouForbidden %s", 403, payload)
}

func (o *GetUsersSharedWithYouForbidden) String() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[GET /api/users/shared-with-you][%d] getUsersSharedWithYouForbidden %s", 403, payload)
}

func (o *GetUsersSharedWithYouForbidden) GetPayload() *GetUsersSharedWithYouForbiddenBody {
	return o.Payload
}

func (o *GetUsersSharedWithYouForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(GetUsersSharedWithYouForbiddenBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetUsersSharedWithYouInternalServerError creates a GetUsersSharedWithYouInternalServerError with default headers values
func NewGetUsersSharedWithYouInternalServerError() *GetUsersSharedWithYouInternalServerError {
	return &GetUsersSharedWithYouInternalServerError{}
}

/*
GetUsersSharedWithYouInternalServerError describes a response with status code 500, with default header values.

Request failed. Internal server error.
*/
type GetUsersSharedWithYouInternalServerError struct {
	Payload *GetUsersSharedWithYouInternalServerErrorBody
}

// IsSuccess returns true when this get users shared with you internal server error response has a 2xx status code
func (o *GetUsersSharedWithYouInternalServerError) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this get users shared with you internal server error response has a 3xx status code
func (o *GetUsersSharedWithYouInternalServerError) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get users shared with you internal server error response has a 4xx status code
func (o *GetUsersSharedWithYouInternalServerError) IsClientError() bool {
	return false
}

// IsServerError returns true when this get users shared with you internal server error response has a 5xx status code
func (o *GetUsersSharedWithYouInternalServerError) IsServerError() bool {
	return true
}

// IsCode returns true when this get users shared with you internal server error response a status code equal to that given
func (o *GetUsersSharedWithYouInternalServerError) IsCode(code int) bool {
	return code == 500
}

// Code gets the status code for the get users shared with you internal server error response
func (o *GetUsersSharedWithYouInternalServerError) Code() int {
	return 500
}

func (o *GetUsersSharedWithYouInternalServerError) Error() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[GET /api/users/shared-with-you][%d] getUsersSharedWithYouInternalServerError %s", 500, payload)
}

func (o *GetUsersSharedWithYouInternalServerError) String() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[GET /api/users/shared-with-you][%d] getUsersSharedWithYouInternalServerError %s", 500, payload)
}

func (o *GetUsersSharedWithYouInternalServerError) GetPayload() *GetUsersSharedWithYouInternalServerErrorBody {
	return o.Payload
}

func (o *GetUsersSharedWithYouInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(GetUsersSharedWithYouInternalServerErrorBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

/*
GetUsersSharedWithYouForbiddenBody get users shared with you forbidden body
swagger:model GetUsersSharedWithYouForbiddenBody
*/
type GetUsersSharedWithYouForbiddenBody struct {

	// message
	Message string `json:"message,omitempty"`
}

// Validate validates this get users shared with you forbidden body
func (o *GetUsersSharedWithYouForbiddenBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this get users shared with you forbidden body based on context it is used
func (o *GetUsersSharedWithYouForbiddenBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *GetUsersSharedWithYouForbiddenBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *GetUsersSharedWithYouForbiddenBody) UnmarshalBinary(b []byte) error {
	var res GetUsersSharedWithYouForbiddenBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
GetUsersSharedWithYouInternalServerErrorBody get users shared with you internal server error body
swagger:model GetUsersSharedWithYouInternalServerErrorBody
*/
type GetUsersSharedWithYouInternalServerErrorBody struct {

	// message
	Message string `json:"message,omitempty"`
}

// Validate validates this get users shared with you internal server error body
func (o *GetUsersSharedWithYouInternalServerErrorBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this get users shared with you internal server error body based on context it is used
func (o *GetUsersSharedWithYouInternalServerErrorBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *GetUsersSharedWithYouInternalServerErrorBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *GetUsersSharedWithYouInternalServerErrorBody) UnmarshalBinary(b []byte) error {
	var res GetUsersSharedWithYouInternalServerErrorBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
GetUsersSharedWithYouOKBody get users shared with you o k body
swagger:model GetUsersSharedWithYouOKBody
*/
type GetUsersSharedWithYouOKBody struct {

	// users
	Users []*GetUsersSharedWithYouOKBodyUsersItems0 `json:"users"`
}

// Validate validates this get users shared with you o k body
func (o *GetUsersSharedWithYouOKBody) Validate(formats strfmt.Registry) error {
	var res []error

	if err := o.validateUsers(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *GetUsersSharedWithYouOKBody) validateUsers(formats strfmt.Registry) error {
	if swag.IsZero(o.Users) { // not required
		return nil
	}

	for i := 0; i < len(o.Users); i++ {
		if swag.IsZero(o.Users[i]) { // not required
			continue
		}

		if o.Users[i] != nil {
			if err := o.Users[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("getUsersSharedWithYouOK" + "." + "users" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("getUsersSharedWithYouOK" + "." + "users" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// ContextValidate validate this get users shared with you o k body based on the context it is used
func (o *GetUsersSharedWithYouOKBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := o.contextValidateUsers(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *GetUsersSharedWithYouOKBody) contextValidateUsers(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(o.Users); i++ {

		if o.Users[i] != nil {

			if swag.IsZero(o.Users[i]) { // not required
				return nil
			}

			if err := o.Users[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("getUsersSharedWithYouOK" + "." + "users" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("getUsersSharedWithYouOK" + "." + "users" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// MarshalBinary interface implementation
func (o *GetUsersSharedWithYouOKBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *GetUsersSharedWithYouOKBody) UnmarshalBinary(b []byte) error {
	var res GetUsersSharedWithYouOKBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
GetUsersSharedWithYouOKBodyUsersItems0 get users shared with you o k body users items0
swagger:model GetUsersSharedWithYouOKBodyUsersItems0
*/
type GetUsersSharedWithYouOKBodyUsersItems0 struct {

	// email
	Email string `json:"email,omitempty"`
}

// Validate validates this get users shared with you o k body users items0
func (o *GetUsersSharedWithYouOKBodyUsersItems0) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this get users shared with you o k body users items0 based on context it is used
func (o *GetUsersSharedWithYouOKBodyUsersItems0) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *GetUsersSharedWithYouOKBodyUsersItems0) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *GetUsersSharedWithYouOKBodyUsersItems0) UnmarshalBinary(b []byte) error {
	var res GetUsersSharedWithYouOKBodyUsersItems0
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}

/*
GetUsersSharedWithYouUnauthorizedBody get users shared with you unauthorized body
swagger:model GetUsersSharedWithYouUnauthorizedBody
*/
type GetUsersSharedWithYouUnauthorizedBody struct {

	// message
	Message string `json:"message,omitempty"`
}

// Validate validates this get users shared with you unauthorized body
func (o *GetUsersSharedWithYouUnauthorizedBody) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this get users shared with you unauthorized body based on context it is used
func (o *GetUsersSharedWithYouUnauthorizedBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *GetUsersSharedWithYouUnauthorizedBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *GetUsersSharedWithYouUnauthorizedBody) UnmarshalBinary(b []byte) error {
	var res GetUsersSharedWithYouUnauthorizedBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}
