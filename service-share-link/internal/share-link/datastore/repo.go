package datastore

import (
	"context"
	"errors"
	"fmt"
	"service-share-link/internal/share-link/constants"
	"service-share-link/internal/share-link/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ShareLinkRepo struct {
	pool *pgxpool.Pool
}

type Repository interface {
	FindByCode(ctx context.Context, code model.Code) (*model.ShareLink, error)
	FindLiveByResource(ctx context.Context, resourceId model.ResourceId) (*model.ShareLink, error)
	UpdateStatus(ctx context.Context, rc model.ShareLink) (*model.ShareLink, error)
	Insert(ctx context.Context, rc model.ShareLink) (*model.ShareLink, error)
	IncrementViewCounts(ctx context.Context, deltas map[model.Code]int64) error
}

func NewRepoShareLink(pool *pgxpool.Pool) *ShareLinkRepo {
	return &ShareLinkRepo{
		pool: pool,
	}
}

func (repo *ShareLinkRepo) FindByCode(ctx context.Context, code model.Code) (*model.ShareLink, error) {
	row := repo.pool.QueryRow(ctx, `
		SELECT id, code, owner_id, resource_id, status, view_count, created_at, updated_at
		FROM share_links
		WHERE code = $1
	`, code)

	return queryException(row)
}

// FindLiveByResource returns the link that is not deleted yet. There is at most
// one, enforced by share_links_live_resource_key.
func (repo *ShareLinkRepo) FindLiveByResource(ctx context.Context, resourceId model.ResourceId) (*model.ShareLink, error) {
	row := repo.pool.QueryRow(ctx, `
		SELECT id, code, owner_id, resource_id, status, view_count, created_at, updated_at
		FROM share_links
		WHERE resource_id = $1
		  AND status <> 'deleted'
	`, resourceId)

	return queryException(row)
}

func (repo *ShareLinkRepo) UpdateStatus(ctx context.Context, rc model.ShareLink) (*model.ShareLink, error) {
	row := repo.pool.QueryRow(ctx, `
		UPDATE share_links
		SET status 		= $1,
			updated_at	= now(),
			deleted_at	= CASE WHEN $1 = 'deleted' THEN now() END
		WHERE code		= $2
		  AND status	<> 'deleted'
		RETURNING id, code, owner_id, resource_id, status, view_count, created_at, updated_at
	`, string(rc.Status), rc.Code)

	rowUpdate, err := queryScanRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, constants.ErrorRecordDeleteNotChange
	}
	if err != nil {
		return nil, fmt.Errorf("[Query] Record update failed %w", err)
	}

	return rowUpdate, nil
}

func (repo *ShareLinkRepo) Insert(ctx context.Context, link model.ShareLink) (*model.ShareLink, error) {
	row := repo.pool.QueryRow(ctx, `
		INSERT INTO share_links (code, owner_id, resource_id, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, code, owner_id, resource_id, status, view_count, created_at, updated_at
	`, link.Code, link.OwnerId, link.ResourceId, link.Status)

	record, err := queryScanRow(row)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "share_links_code_key":
			return nil, constants.ErrQueryConflict
		case "share_links_live_resource_key":
			return nil, constants.ErrQueryResourceConflict
		}
	}
	if err != nil {
		return nil, fmt.Errorf("[Query] Insert share link %w", err)
	}
	return record, nil
}

// IncrementViewCounts applies a whole batch in one statement, so a hot link
// costs one row update per flush instead of one per view.
func (repo *ShareLinkRepo) IncrementViewCounts(ctx context.Context, deltas map[model.Code]int64) error {
	codes := make([]string, 0, len(deltas))
	counts := make([]int64, 0, len(deltas))
	for code, delta := range deltas {
		codes = append(codes, string(code))
		counts = append(counts, delta)
	}

	_, err := repo.pool.Exec(ctx, `
		UPDATE share_links AS s
		SET view_count = s.view_count + v.delta
		FROM unnest($1::text[], $2::bigint[]) AS v(code, delta)
		WHERE s.code = v.code
	`, codes, counts)
	if err != nil {
		return fmt.Errorf("[Query] Increment view counts %w", err)
	}

	return nil
}

func queryScanRow(row pgx.Row) (*model.ShareLink, error) {
	var record model.ShareLink
	err := row.Scan(
		&record.Id, &record.Code,
		&record.OwnerId, &record.ResourceId,
		&record.Status, &record.ViewCount,
		&record.CreatedAt, &record.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func queryException(row pgx.Row) (*model.ShareLink, error) {
	record, err := queryScanRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("[Query] Scan link share %w", err)
	}

	return record, nil
}
