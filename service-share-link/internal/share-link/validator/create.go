package validator

import (
	"errors"
	"service-share-link/internal/share-link/model"
)

type CreateShareRequest struct {
	ResourceId model.ResourceId `json:"resource_id"`
}

func ValidateCreateShare(body *CreateShareRequest) error {
	if body.ResourceId <= 0 {
		return errors.New("resource_id must be a positive integer")
	}

	return nil
}
