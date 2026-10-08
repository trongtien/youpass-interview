package service

import (
	"context"
	"log"
	"service-share-link/internal/share-link/constants"
	"service-share-link/internal/share-link/model"
)

func (sv *ShareLinkService) RecordView(ctx context.Context, code model.Code) error {
	if _, err := sv.GetLinkShare(ctx, code); err != nil {
		return err
	}

	if err := sv.views.Incr(ctx, code); err != nil {
		log.Printf("[Service] Incr view code=%s: %v", code, err)
		return constants.ErrorSever
	}

	return nil
}
