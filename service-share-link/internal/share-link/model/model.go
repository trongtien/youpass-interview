package model

import (
	"service-share-link/internal/share-link/constants"
	"time"
)

type Status string
type Code string

const (
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
	StatusDelete   Status = "deleted"
)

type ResourceId int64
type OwnerId int64

type ShareLink struct {
	Id         string
	Code       Code
	Status     Status
	OwnerId    OwnerId
	ResourceId ResourceId
	ViewCount  int
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  time.Time
}

func NewShareLink(code Code, ownerId OwnerId, resourceId ResourceId) ShareLink {
	return ShareLink{
		Code:       code,
		OwnerId:    ownerId,
		ResourceId: resourceId,
		Status:     StatusActive,
	}
}

func (slm *ShareLink) IsEnabled() bool {
	return slm.Status == StatusActive
}

func (slm *ShareLink) SetStatusEnabled(enabled bool) (*ShareLink, error) {
	if slm.Status == StatusDelete {
		return nil, constants.ErrorRecordDeleteNotChange
	}

	currentDefault := StatusDisabled
	if enabled {
		currentDefault = StatusActive
	}

	if slm.Status == currentDefault {
		return slm, nil
	}

	slm.Status = currentDefault

	return slm, nil
}

func (slm *ShareLink) Delete() (*ShareLink, bool) {
	if slm.Status == StatusDelete {
		return slm, false
	}

	slm.Status = StatusDelete

	return slm, true
}

func (slm *ShareLink) IsViewEnable() error {
	if slm.Status == StatusDelete {
		return constants.ErrLinkDeleted
	}

	if slm.Status == StatusDisabled {
		return constants.ErrLinkDisable
	}

	return nil
}
