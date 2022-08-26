// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// CheckResults check results
//
// swagger:model CheckResults
type CheckResults struct {

	// pod results
	PodResults map[string]PodResult `json:"podResults,omitempty"`

	// probe results
	ProbeResults ProbeResults `json:"probeResults,omitempty"`
}

// Validate validates this check results
func (m *CheckResults) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validatePodResults(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateProbeResults(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *CheckResults) validatePodResults(formats strfmt.Registry) error {
	if swag.IsZero(m.PodResults) { // not required
		return nil
	}

	for k := range m.PodResults {

		if err := validate.Required("podResults"+"."+k, "body", m.PodResults[k]); err != nil {
			return err
		}
		if val, ok := m.PodResults[k]; ok {
			if err := val.Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("podResults" + "." + k)
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("podResults" + "." + k)
				}
				return err
			}
		}

	}

	return nil
}

func (m *CheckResults) validateProbeResults(formats strfmt.Registry) error {
	if swag.IsZero(m.ProbeResults) { // not required
		return nil
	}

	if m.ProbeResults != nil {
		if err := m.ProbeResults.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("probeResults")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("probeResults")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this check results based on the context it is used
func (m *CheckResults) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidatePodResults(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateProbeResults(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *CheckResults) contextValidatePodResults(ctx context.Context, formats strfmt.Registry) error {

	for k := range m.PodResults {

		if val, ok := m.PodResults[k]; ok {
			if err := val.ContextValidate(ctx, formats); err != nil {
				return err
			}
		}

	}

	return nil
}

func (m *CheckResults) contextValidateProbeResults(ctx context.Context, formats strfmt.Registry) error {

	if err := m.ProbeResults.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("probeResults")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("probeResults")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *CheckResults) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *CheckResults) UnmarshalBinary(b []byte) error {
	var res CheckResults
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
