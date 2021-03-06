// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/bloomberg/goldpinger/v3/pkg/models"
)

// ClusterHealthReader is a Reader for the ClusterHealth structure.
type ClusterHealthReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ClusterHealthReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewClusterHealthOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 418:
		result := NewClusterHealthIMATeapot()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewClusterHealthOK creates a ClusterHealthOK with default headers values
func NewClusterHealthOK() *ClusterHealthOK {
	return &ClusterHealthOK{}
}

/* ClusterHealthOK describes a response with status code 200, with default header values.

Healthy cluster
*/
type ClusterHealthOK struct {
	Payload *models.ClusterHealthResults
}

func (o *ClusterHealthOK) Error() string {
	return fmt.Sprintf("[GET /cluster_health][%d] clusterHealthOK  %+v", 200, o.Payload)
}
func (o *ClusterHealthOK) GetPayload() *models.ClusterHealthResults {
	return o.Payload
}

func (o *ClusterHealthOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ClusterHealthResults)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewClusterHealthIMATeapot creates a ClusterHealthIMATeapot with default headers values
func NewClusterHealthIMATeapot() *ClusterHealthIMATeapot {
	return &ClusterHealthIMATeapot{}
}

/* ClusterHealthIMATeapot describes a response with status code 418, with default header values.

Unhealthy cluster
*/
type ClusterHealthIMATeapot struct {
	Payload *models.ClusterHealthResults
}

func (o *ClusterHealthIMATeapot) Error() string {
	return fmt.Sprintf("[GET /cluster_health][%d] clusterHealthIMATeapot  %+v", 418, o.Payload)
}
func (o *ClusterHealthIMATeapot) GetPayload() *models.ClusterHealthResults {
	return o.Payload
}

func (o *ClusterHealthIMATeapot) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ClusterHealthResults)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
