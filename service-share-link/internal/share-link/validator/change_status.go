package validator

import "errors"

type ChangeStatusRequest struct {
	Enabled *bool `json:"enabled"`
}

func ValidateChangeStatus(body *ChangeStatusRequest) error {
	if body.Enabled == nil {
		return errors.New("enabled is required")
	}

	return nil
}
