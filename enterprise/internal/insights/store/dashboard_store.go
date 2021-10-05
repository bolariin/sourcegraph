package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
)

type DashboardStore struct {
	*basestore.Store
	Now func() time.Time
}

// NewDashboardStore returns a new DashboardStore backed by the given Timescale db.
func NewDashboardStore(db dbutil.DB) *DashboardStore {
	return &DashboardStore{Store: basestore.NewWithDB(db, sql.TxOptions{}), Now: time.Now}
}

// Handle returns the underlying transactable database handle.
// Needed to implement the ShareableStore interface.
func (s *DashboardStore) Handle() *basestore.TransactableHandle { return s.Store.Handle() }

// With creates a new DashboardStore with the given basestore. Shareable store as the underlying basestore.Store.
// Needed to implement the basestore.Store interface
func (s *DashboardStore) With(other *DashboardStore) *DashboardStore {
	return &DashboardStore{Store: s.Store.With(other.Store), Now: other.Now}
}

func (s *DashboardStore) Transact(ctx context.Context) (*DashboardStore, error) {
	txBase, err := s.Store.Transact(ctx)
	return &DashboardStore{Store: txBase, Now: s.Now}, err
}

func (s *DashboardStore) DeleteDashboard(ctx context.Context, id string) error {
	const deleteDashboardSql = `
	-- source: enterprise/internal/insights/store/dashboard_store.go:DeleteDashboard
	update dashboard set deleted_at = NOW() where id = %s;
	`

	err := s.Exec(ctx, sqlf.Sprintf(deleteDashboardSql, id))
	if err != nil {
		return errors.Wrapf(err, "failed to delete dashboard with id: %s", id)
	}
	return nil
}
