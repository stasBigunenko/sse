package postgres

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"sse/config"
)

type Postgres struct {
	db *pgxpool.Pool
	mu sync.Mutex
}

var (
	pgInstance *Postgres
	pgOnce     sync.Once
)

func NewPG(ctx context.Context, postgresCfg config.PostgresConfig) (*Postgres, error) {
	var (
		db  *pgxpool.Pool
		err error
	)

	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		postgresCfg.User,
		postgresCfg.Password,
		postgresCfg.Host,
		postgresCfg.Port,
		postgresCfg.DBName,
	)

	pgOnce.Do(func() {
		db, err = pgxpool.New(ctx, connString)
		if err != nil {
			err = fmt.Errorf("unable to create connection pool: %w", err)
			return
		}

		pgInstance = &Postgres{db: db}
	})

	if err != nil {
		return nil, err
	}

	return pgInstance, nil
}

func (p *Postgres) Ping(ctx context.Context) error {
	return p.db.Ping(ctx)
}

func (p *Postgres) Close() {
	p.db.Close()
}
