package service

import (
	"context"
	"errors"
	"log"
	"service-share-link/internal/share-link/constants"
	"service-share-link/internal/share-link/datastore"
	"service-share-link/internal/share-link/model"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	MAX_RETRY_CREATE_ATTEMPTS = 3

	loadFromDBTimeout = 3 * time.Second
	saveStatusTimeout = 3 * time.Second
)

type ServiceShareLink interface {
	GetLinkShare(ctx context.Context, code model.Code) (*model.ShareLink, error)
	RecordView(ctx context.Context, code model.Code) error
	CreateLinkShare(ctx context.Context, ownerId model.OwnerId, resourceId model.ResourceId) (link *model.ShareLink, created bool, err error)
	ChangeStatusLinkShare(ctx context.Context, ownerId model.OwnerId, code model.Code, enabled bool) (*model.ShareLink, error)
	DeleteLinkShare(ctx context.Context, ownerId model.OwnerId, code model.Code) error
}

type ShareLinkService struct {
	repo  datastore.Repository
	cache datastore.Cache
	views datastore.ViewCounter
	group singleflight.Group
}

func NewService(repo datastore.Repository, cache datastore.Cache, views datastore.ViewCounter) *ShareLinkService {
	return &ShareLinkService{
		repo:  repo,
		cache: cache,
		views: views,
	}
}

func (sv *ShareLinkService) GetLinkShare(ctx context.Context, code model.Code) (*model.ShareLink, error) {
	linkShare, err := sv.findByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	if err := linkShare.IsViewEnable(); err != nil {
		return nil, err
	}

	return linkShare, nil
}

func (sv *ShareLinkService) findByCode(ctx context.Context, code model.Code) (*model.ShareLink, error) {
	linkShare, hit, err := sv.cache.Get(ctx, code)
	if err != nil {
		log.Printf("[Service] Cache get code=%s: %v", code, err)
	}
	if hit {
		if linkShare == nil {
			return nil, constants.ErrorRecordNotFound
		}
		return linkShare, nil
	}

	result, err, _ := sv.group.Do(string(code), func() (any, error) {
		return sv.loadFromDB(context.WithoutCancel(ctx), code)
	})
	if err != nil {
		return nil, err
	}

	loaded := *result.(*model.ShareLink)
	return &loaded, nil
}

func (sv *ShareLinkService) loadFromDB(ctx context.Context, code model.Code) (*model.ShareLink, error) {
	ctx, cancel := context.WithTimeout(ctx, loadFromDBTimeout)
	defer cancel()

	linkShare, err := sv.repo.FindByCode(ctx, code)
	if err != nil {
		log.Printf("[Service] Find share link code=%s: %v", code, err)
		return nil, constants.ErrorSever
	}

	if linkShare == nil {
		if err := sv.cache.SetNotFound(ctx, code); err != nil {
			log.Printf("[Service] Cache set not found code=%s: %v", code, err)
		}
		return nil, constants.ErrorRecordNotFound
	}

	if err := sv.cache.SetIfAbsent(ctx, linkShare); err != nil {
		log.Printf("[Service] Cache set code=%s: %v", code, err)
	}

	return linkShare, nil
}

func (sv *ShareLinkService) ChangeStatusLinkShare(ctx context.Context, ownerId model.OwnerId, code model.Code, enabled bool) (*model.ShareLink, error) {
	linkShare, err := sv.findOwnedByCode(ctx, ownerId, code)
	if err != nil {
		return nil, err
	}

	current := linkShare.Status
	if _, err := linkShare.SetStatusEnabled(enabled); err != nil {
		return nil, err
	}
	if linkShare.Status == current {
		return linkShare, nil
	}

	return sv.saveStatus(ctx, linkShare)
}

func (sv *ShareLinkService) DeleteLinkShare(ctx context.Context, ownerId model.OwnerId, code model.Code) error {
	linkShare, err := sv.findOwnedByCode(ctx, ownerId, code)
	if err != nil {
		return err
	}

	if _, changed := linkShare.Delete(); !changed {
		return nil
	}

	_, err = sv.saveStatus(ctx, linkShare)
	if errors.Is(err, constants.ErrorRecordDeleteNotChange) {
		return nil
	}

	return err
}

func (sv *ShareLinkService) findOwnedByCode(ctx context.Context, ownerId model.OwnerId, code model.Code) (*model.ShareLink, error) {
	linkShare, err := sv.repo.FindByCode(ctx, code)
	if err != nil {
		log.Printf("[Service] Find share link code=%s: %v", code, err)
		return nil, constants.ErrorSever
	}

	if linkShare == nil {
		return nil, constants.ErrorRecordNotFound
	}

	if linkShare.OwnerId != ownerId {
		return nil, constants.ErrNotOwner
	}

	return linkShare, nil
}

func (sv *ShareLinkService) saveStatus(ctx context.Context, linkShare *model.ShareLink) (*model.ShareLink, error) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), saveStatusTimeout)
	defer cancel()

	updated, err := sv.repo.UpdateStatus(ctx, *linkShare)
	if errors.Is(err, constants.ErrorRecordDeleteNotChange) {
		return nil, err
	}
	if err != nil {
		log.Printf("[Service] Update status code=%s: %v", linkShare.Code, err)
		sv.invalidateCache(ctx, linkShare.Code)
		return nil, constants.ErrorSever
	}

	if err := sv.cache.Set(ctx, updated); err != nil {
		log.Printf("[Service] Cache set code=%s: %v", updated.Code, err)
		sv.invalidateCache(ctx, updated.Code)
	}

	return updated, nil
}

func (sv *ShareLinkService) invalidateCache(ctx context.Context, code model.Code) {
	if err := sv.cache.Delete(ctx, code); err != nil {
		log.Printf("[Service] Cache delete code=%s, entry stays until TTL: %v", code, err)
	}
}

func (sv *ShareLinkService) CreateLinkShare(ctx context.Context, ownerId model.OwnerId, resourceId model.ResourceId) (*model.ShareLink, bool, error) {
	for range MAX_RETRY_CREATE_ATTEMPTS {
		code, err := GeneratorCode()
		if err != nil {
			log.Printf("[Service] Generate code: %v", err)
			return nil, false, constants.ErrorSever
		}

		linkShare, err := sv.repo.Insert(ctx, model.NewShareLink(code, ownerId, resourceId))
		if errors.Is(err, constants.ErrQueryConflict) {
			continue
		}
		if errors.Is(err, constants.ErrQueryResourceConflict) {
			existing, err := sv.findLiveOwnedByResource(ctx, ownerId, resourceId)
			if err != nil {
				return nil, false, err
			}

			if existing == nil {
				continue
			}
			return existing, false, nil
		}
		if err != nil {
			log.Printf("[Service] Insert share link owner=%d resource=%d: %v", ownerId, resourceId, err)
			return nil, false, constants.ErrorSever
		}

		if err := sv.cache.Set(ctx, linkShare); err != nil {
			log.Printf("[Service] Cache set code=%s: %v", linkShare.Code, err)
		}

		return linkShare, true, nil
	}

	return nil, false, constants.ErrorSever
}

func (sv *ShareLinkService) findLiveOwnedByResource(ctx context.Context, ownerId model.OwnerId, resourceId model.ResourceId) (*model.ShareLink, error) {
	existing, err := sv.repo.FindLiveByResource(ctx, resourceId)
	if err != nil {
		log.Printf("[Service] Find live link resource=%d: %v", resourceId, err)
		return nil, constants.ErrorSever
	}

	if existing != nil && existing.OwnerId != ownerId {
		return nil, constants.ErrResourceSharedByOther
	}

	return existing, nil
}
