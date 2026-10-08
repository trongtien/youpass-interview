package service

import (
	"context"
	"fmt"
	"log"
	"service-share-link/internal/share-link/datastore"
	"time"
)

const (
	ViewFlushInterval = 10 * time.Second
	viewFlushTimeout  = 5 * time.Second
	viewAckTimeout    = 2 * time.Second

	viewStaleAfter      = 1 * time.Minute
	viewStaleBatchLimit = 10
)

type ViewFlusher struct {
	repo     datastore.Repository
	views    datastore.ViewCounter
	interval time.Duration
}

func NewViewFlusher(repo datastore.Repository, views datastore.ViewCounter, interval time.Duration) *ViewFlusher {
	return &ViewFlusher{
		repo:     repo,
		views:    views,
		interval: interval,
	}
}

func (svv *ViewFlusher) Run(ctx context.Context) {
	ticker := time.NewTicker(svv.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			svv.flushAndLog(ctx)
		case <-ctx.Done():
			svv.flushAndLog(context.WithoutCancel(ctx))
			return
		}
	}
}

func (svv *ViewFlusher) Flush(ctx context.Context) error {
	flushCtx, cancel := context.WithTimeout(ctx, viewFlushTimeout)
	defer cancel()

	stale, err := svv.views.ClaimStale(flushCtx, viewStaleAfter, viewStaleBatchLimit)
	if err != nil {
		return err
	}
	for i, batch := range stale {
		if err := svv.write(ctx, flushCtx, batch); err != nil {
			svv.release(ctx, stale[i+1:]...)
			return err
		}
	}

	batch, err := svv.views.Claim(flushCtx)
	if err != nil || batch == nil {
		return err
	}

	return svv.write(ctx, flushCtx, batch)
}

func (svv *ViewFlusher) write(ctx, flushCtx context.Context, batch *datastore.ViewBatch) error {
	if err := svv.repo.IncrementViewCounts(flushCtx, batch.Deltas); err != nil {
		svv.release(ctx, batch)
		return err
	}

	ackCtx, cancel := newAckContext(ctx)
	defer cancel()
	ack, err := svv.views.Ack(ackCtx, batch.Key)
	if err != nil {
		return fmt.Errorf("batch written but not ack, it will be written again once stale: %w", err)
	}
	if !ack {
		log.Printf("[ViewFlusher] Batch %s was taken over before ack, its %d links are written twice", batch.Key, len(batch.Deltas))
	}

	return nil
}

func (svv *ViewFlusher) release(ctx context.Context, batches ...*datastore.ViewBatch) {
	ackCtx, cancel := newAckContext(ctx)
	defer cancel()

	for _, batch := range batches {
		if err := svv.views.Release(ackCtx, batch.Key); err != nil {
			log.Printf("[ViewFlusher] Release batch %s, retried once stale: %v", batch.Key, err)
		}
	}
}

func (svv *ViewFlusher) flushAndLog(ctx context.Context) {
	if err := svv.Flush(ctx); err != nil {
		log.Printf("[ViewFlusher] Flush views: %v", err)
	}
}

func newAckContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), viewAckTimeout)
}
