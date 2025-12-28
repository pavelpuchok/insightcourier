package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pavelpuchok/insightcourier/config"
	"github.com/pavelpuchok/insightcourier/storage/psql"
)

type PostgreSQL struct {
	conn    *pgx.Conn
	timeout time.Duration
}

func NewPostgreSQL(ctx context.Context, config config.PSQLStorageConfig) (*PostgreSQL, error) {
	conn, err := pgx.Connect(ctx, config.ConnString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL DB. %w", err)
	}

	return &PostgreSQL{
		conn:    conn,
		timeout: config.DefaultTimeout,
	}, nil
}

type postgreSQLTxKeyType string

var postgreSQLTxKey = postgreSQLTxKeyType("psql_tx")

func (pq *PostgreSQL) BeginTxInContext(ctx context.Context) (context.Context, error) {
	tx, err := pq.conn.Begin(ctx)
	if err != nil {
		return ctx, fmt.Errorf("fail to begin transaction. %w", err)
	}
	return context.WithValue(ctx, postgreSQLTxKey, tx), nil
}

func (pq *PostgreSQL) CommitTxInContext(ctx context.Context) error {
	tx, ok := ctx.Value(postgreSQLTxKey).(pgx.Tx)
	if !ok {
		return errors.New("fail to commit transaction. transcation not found in context")
	}
	return tx.Commit(ctx)
}

func (pq *PostgreSQL) RollbackTxInContext(ctx context.Context) error {
	tx, ok := ctx.Value(postgreSQLTxKey).(pgx.Tx)
	if !ok {
		return errors.New("fail to rollback transaction. transcation not found in context")
	}
	return tx.Rollback(ctx)
}

func (pq *PostgreSQL) CreateSource(ctx context.Context, source string) (int32, error) {
	q := pq.getQueriesFromContext(ctx)
	cctx, cancel := context.WithTimeout(ctx, pq.timeout)
	defer cancel()

	id, err := q.CreateSource(cctx, psql.CreateSourceParams{
		Name:      source,
		CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
	})

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return 0, ErrSourceAlreadyExists
			}
		}
		return 0, fmt.Errorf("failed to create source (%s). %w", source, err)
	}

	return id, nil
}

func (pq *PostgreSQL) getQueriesFromContext(ctx context.Context) *psql.Queries {
	q := psql.New(pq.conn)
	tx, ok := ctx.Value(postgreSQLTxKey).(pgx.Tx)
	if !ok {
		return q
	}

	return q.WithTx(tx)
}

func (pq *PostgreSQL) GetSourceUpdateTime(ctx context.Context, source string) (*time.Time, error) {
	q := pq.getQueriesFromContext(ctx)
	cctx, cancel := context.WithTimeout(ctx, pq.timeout)
	defer cancel()

	t, err := q.GetSourceLastFetchedAtByName(cctx, source)
	if err != nil {
		return nil, fmt.Errorf("failed to get sources (%s) last fetched time. %w", source, err)
	}
	return &t.Time, nil
}

func (pq *PostgreSQL) SetSourceUpdateTime(ctx context.Context, source string, t time.Time) error {
	q := pq.getQueriesFromContext(ctx)
	cctx, cancel := context.WithTimeout(ctx, pq.timeout)
	defer cancel()

	err := q.SetSourceLastFetchedAtByName(cctx, psql.SetSourceLastFetchedAtByNameParams{
		Name:          source,
		LastFetchedAt: pgtype.Timestamp{Time: t, Valid: true},
	})

	if err != nil {
		return fmt.Errorf("failed to set sources (%s) fetch time. %w", source, err)
	}

	return nil
}
