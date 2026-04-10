package transactionbus

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/semaphore"

	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/foundation/logger"
)

type Storer interface {
	Create(ctx context.Context, t Transaction) error
	CreateBatch(ctx context.Context, txns []Transaction) (int, error)
	Update(ctx context.Context, t Transaction) error
	Delete(ctx context.Context, t Transaction) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Transaction, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (Transaction, error)
}

const (
	enrichBatchTimeout  = 2 * time.Minute
	maxConcurrentEnrich = 3
)

type Business struct {
	log          *logger.Logger
	storer       Storer
	enricher     Enricher
	enrichSem    *semaphore.Weighted
	enrichActive   atomic.Int64
	enrichPending  atomic.Int64
	enrichDone     atomic.Int64
	enrichFailed   atomic.Int64
}

func NewBusiness(log *logger.Logger, storer Storer) *Business {
	return &Business{
		log:       log,
		storer:    storer,
		enrichSem: semaphore.NewWeighted(maxConcurrentEnrich),
	}
}

// WithEnricher sets an optional enricher for async transaction enrichment.
func (b *Business) WithEnricher(e Enricher) {
	b.enricher = e
}

type EnrichmentStatus struct {
	Active  int64
	Pending int64
	Done    int64
	Failed  int64
	Enabled bool
}

func (b *Business) EnrichmentStatus() EnrichmentStatus {
	return EnrichmentStatus{
		Active:  b.enrichActive.Load(),
		Pending: b.enrichPending.Load(),
		Done:    b.enrichDone.Load(),
		Failed:  b.enrichFailed.Load(),
		Enabled: b.enricher != nil,
	}
}

func (b *Business) Create(ctx context.Context, nt NewTransaction) (Transaction, error) {
	now := time.Now()

	t := Transaction{
		ID:          uuid.New(),
		RawInputID:  nt.RawInputID,
		Source:      nt.Source,
		Date:        nt.Date,
		Description: nt.Description,
		CleanName:   nt.CleanName,
		Amount:      nt.Amount,
		Category:    nt.Category,
		ContextID:   nt.ContextID,
		Notes:       nt.Notes,
		Reviewed:    false,
		CreatedAt:   now,
	}

	if err := b.storer.Create(ctx, t); err != nil {
		return Transaction{}, fmt.Errorf("create: %w", err)
	}

	return t, nil
}

func (b *Business) CreateBatch(ctx context.Context, nts []NewTransaction) (int, error) {
	now := time.Now()

	txns := make([]Transaction, len(nts))
	for i, nt := range nts {
		txns[i] = Transaction{
			ID:          uuid.New(),
			RawInputID:  nt.RawInputID,
			Source:      nt.Source,
			Date:        nt.Date,
			Description: nt.Description,
			CleanName:   nt.CleanName,
			Amount:      nt.Amount,
			Category:    nt.Category,
			ContextID:   nt.ContextID,
			Notes:       nt.Notes,
			Reviewed:    false,
			CreatedAt:   now,
		}
	}

	inserted, err := b.storer.CreateBatch(ctx, txns)
	if err != nil {
		return 0, fmt.Errorf("create batch: %w", err)
	}

	if b.enricher != nil {
		b.enrichPending.Add(int64(len(txns)))
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), enrichBatchTimeout)
			defer cancel()

			if err := b.enrichSem.Acquire(ctx, 1); err != nil {
				count := int64(len(txns))
				b.enrichPending.Add(-count)
				b.enrichFailed.Add(count)
				b.log.Error(ctx, "transaction_enrichment", "status", "cancelled waiting for semaphore", "error", err.Error())
				return
			}
			defer b.enrichSem.Release(1)

			b.enrichBatch(ctx, txns)
		}()
	}

	return inserted, nil
}

func (b *Business) Update(ctx context.Context, t Transaction, ut UpdateTransaction) (Transaction, error) {
	if ut.CleanName != nil {
		t.CleanName = ut.CleanName
	}
	if ut.Category != nil {
		t.Category = ut.Category
	}
	if ut.ContextID != nil {
		t.ContextID = ut.ContextID
	}
	if ut.Notes != nil {
		t.Notes = ut.Notes
	}
	if ut.Reviewed != nil {
		t.Reviewed = *ut.Reviewed
	}

	if err := b.storer.Update(ctx, t); err != nil {
		return Transaction{}, fmt.Errorf("update: %w", err)
	}

	return t, nil
}

func (b *Business) Delete(ctx context.Context, t Transaction) error {
	if err := b.storer.Delete(ctx, t); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Transaction, error) {
	txns, err := b.storer.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return txns, nil
}

func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	n, err := b.storer.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}

func (b *Business) QueryByID(ctx context.Context, id uuid.UUID) (Transaction, error) {
	t, err := b.storer.QueryByID(ctx, id)
	if err != nil {
		return Transaction{}, fmt.Errorf("query by id[%s]: %w", id, err)
	}
	return t, nil
}

func (b *Business) enrichBatch(ctx context.Context, txns []Transaction) {
	for _, txn := range txns {
		if txn.CleanName != nil && txn.Category != nil {
			b.enrichPending.Add(-1)
			b.enrichDone.Add(1)
			continue
		}

		b.enrichPending.Add(-1)
		b.enrichActive.Add(1)

		enrichment, err := b.enricher.EnrichTransaction(ctx, txn)
		if err != nil {
			b.enrichActive.Add(-1)
			b.enrichFailed.Add(1)
			b.log.Error(ctx, "transaction_enrichment", "status", "failed", "txn_id", txn.ID, "error", err.Error())
			continue
		}

		ut := UpdateTransaction{}
		if txn.CleanName == nil && enrichment.CleanName != "" {
			ut.CleanName = &enrichment.CleanName
		}
		if txn.Category == nil && enrichment.Category != "" {
			ut.Category = &enrichment.Category
		}
		if txn.ContextID == nil && enrichment.SuggestedContextID != nil && enrichment.ContextConfidence >= 0.7 {
			ut.ContextID = enrichment.SuggestedContextID
		}

		if _, err := b.Update(ctx, txn, ut); err != nil {
			b.enrichActive.Add(-1)
			b.enrichFailed.Add(1)
			b.log.Error(ctx, "transaction_enrichment", "status", "update failed", "txn_id", txn.ID, "error", err.Error())
			continue
		}

		b.enrichActive.Add(-1)
		b.enrichDone.Add(1)
	}
}
