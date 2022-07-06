// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// GitlabProjectsReader is a Reader for the GitlabProjects structure.
type GitlabProjectsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GitlabProjectsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGitlabProjectsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 403:
		result := NewGitlabProjectsForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewGitlabProjectsInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewGitlabProjectsOK creates a GitlabProjectsOK with default headers values
func NewGitlabProjectsOK() *GitlabProjectsOK {
	return &GitlabProjectsOK{}
}

/* GitlabProjectsOK describes a response with status code 200, with default header values.

This resource return all projects owned by the user on GitLab in JSON format.
*/
type GitlabProjectsOK struct {
}

func (o *GitlabProjectsOK) Error() string {
	return fmt.Sprintf("[GET /api/gitlab/projects][%d] gitlabProjectsOK ", 200)
}

func (o *GitlabProjectsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGitlabProjectsForbidden creates a GitlabProjectsForbidden with default headers values
func NewGitlabProjectsForbidden() *GitlabProjectsForbidden {
	return &GitlabProjectsForbidden{}
}

/* GitlabProjectsForbidden describes a response with status code 403, with default header values.

Request failed. User token not valid.
*/
type GitlabProjectsForbidden struct {
}

func (o *GitlabProjectsForbidden) Error() string {
	return fmt.Sprintf("[GET /api/gitlab/projects][%d] gitlabProjectsForbidden ", 403)
}

func (o *GitlabProjectsForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGitlabProjectsInternalServerError creates a GitlabProjectsInternalServerError with default headers values
func NewGitlabProjectsInternalServerError() *GitlabProjectsInternalServerError {
	return &GitlabProjectsInternalServerError{}
}

/* GitlabProjectsInternalServerError describes a response with status code 500, with default header values.

Request failed. Internal controller error.
*/
type GitlabProjectsInternalServerError struct {
}

func (o *GitlabProjectsInternalServerError) Error() string {
	return fmt.Sprintf("[GET /api/gitlab/projects][%d] gitlabProjectsInternalServerError ", 500)
}

func (o *GitlabProjectsInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}