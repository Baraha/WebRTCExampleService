package postgresql

import (
	"context"
	"fmt"
	"time"
	"video_service/pkg/logging"
	"video_service/pkg/utils"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type StorageConfig struct {
	Host       string `json:"host"`
	Port       string `json:"port"`
	Database   string `json:"database"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	MaxRetries int    `json:"max_retries"`
}

type Client interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

func NewPostgresClient(ctx context.Context, c StorageConfig) (*pgxpool.Pool, error) {
	var pool *pgxpool.Pool
	var err error
	logger := logging.GetLogger()
	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Database)
	logger.Debugf("dsn is %v", dsn)
	logger.Debug("max_retries %v", c.MaxRetries)
	err = utils.DoWithTries(func() error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		logger.Debug("start connect")
		defer cancel()
		logger.Debug("ctx is %v", ctx)
		pool, err = pgxpool.Connect(ctx, dsn)
		logger.Debugf("err is %v", err)

		if err != nil {
			return err
		}
		return nil
	}, c.MaxRetries, 5*time.Second)

	utils.CatchErr(err)
	logger.Debug("check pool %v", pool)
	return pool, nil
}
