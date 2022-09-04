package video_db_logic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"video_service/internal/adapters/postgresql"
	"video_service/internal/controllers/database/video/video_db_dto"
	"video_service/pkg/logging"
	"video_service/pkg/utils"

	"github.com/jackc/pgconn"
)

type db struct {
	client postgresql.Client
	logger *logging.Logger
}

func NewDBLogic(client postgresql.Client, logger *logging.Logger) db {
	return db{
		client: client,
		logger: logger,
	}
}

func formatQuery(q string) string {
	return strings.ReplaceAll(strings.ReplaceAll(q, "\t", ""), "\n", " ")
}

func (r db) Create(ctx context.Context, video *video_db_dto.Video) error {
	q := `
		INSERT INTO videos 
		    (uri, id, watch_count) 
		VALUES 
		       ($1, $2, $3) 
		RETURNING id
	`
	r.logger.Trace(fmt.Sprintf("SQL Query: %s", formatQuery(q)))
	c := r.client
	r.logger.Debugf("test client %v", c)
	row := r.client.QueryRow(ctx, q, video.Uri, video.ID, video.WatchCount)
	r.logger.Debug("start SCAN")
	if err := row.Scan(&video.ID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			pgErr = err.(*pgconn.PgError)
			newErr := fmt.Errorf(fmt.Sprintf("SQL Error: %s, Detail: %s, Where: %s, Code: %s, SQLState: %s", pgErr.Message, pgErr.Detail, pgErr.Where, pgErr.Code, pgErr.SQLState()))
			r.logger.Error(newErr)
			return newErr
		}
		return err
	}

	return nil
}

func (r db) FindAll(ctx context.Context) (u []video_db_dto.Video, err error) {
	q := `
		SELECT id, uri, watch_count
		 FROM public.videos;
	`
	r.logger.Trace(fmt.Sprintf("SQL Query: %s", formatQuery(q)))

	rows, err := r.client.Query(ctx, q)
	if err != nil {
		return nil, err
	}

	authors := make([]video_db_dto.Video, 0)

	for rows.Next() {
		var v video_db_dto.Video

		err = rows.Scan(&v.ID, &v.Uri, &v.WatchCount)
		if err != nil {
			return nil, err
		}

		authors = append(authors, v)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return authors, nil
}

func (r db) FindOne(ctx context.Context, id string) (video_db_dto.Video, error) {
	q := `
		SELECT id, uri, watch_count FROM public.videos WHERE id = $1
	`
	r.logger.Trace(fmt.Sprintf("SQL Query: %s", formatQuery(q)))

	var ath video_db_dto.Video
	row := r.client.QueryRow(ctx, q, id)
	err := row.Scan(&ath.ID, &ath.Uri, &ath.WatchCount)
	return ath, err
}

func (r db) FindWithUri(ctx context.Context, uri string) (video_db_dto.Video, error) {
	q := `
		SELECT id, uri, watch_count FROM public.videos WHERE uri = $1
	`
	r.logger.Trace(fmt.Sprintf("SQL Query: %s", formatQuery(q)))

	var ath video_db_dto.Video
	row := r.client.QueryRow(ctx, q, uri)
	err := row.Scan(&ath.ID, &ath.Uri, &ath.WatchCount)
	return ath, err
}

func (r db) Delete(ctx context.Context, id string) error {
	q := `
	DELETE FROM public.videos WHERE id = $1
	`
	_, err := r.client.Exec(ctx, q, id)
	utils.CatchErr(err)
	r.logger.Debugf("delete err %v", err)
	return err
}

func (r db) MaxWatch(ctx context.Context) (video_db_dto.Video, error) {
	var max_watch video_db_dto.Video
	q := `
	SELECT id, uri, watch_count
	FROM public.videos 
	WHERE watch_count = (SELECT MAX(watch_count) FROM public.videos)
	`
	row := r.client.QueryRow(ctx, q)
	err := row.Scan(&max_watch.ID, &max_watch.Uri, &max_watch.WatchCount)
	r.logger.Debugf("max err %v", err)

	return max_watch, err
}

func (r db) MinWatch(ctx context.Context) (video_db_dto.Video, error) {
	var min_watch video_db_dto.Video
	q := `
	SELECT id, uri, watch_count
	FROM public.videos 
	WHERE watch_count = (SELECT MIN(watch_count) FROM public.videos) 
	`
	row := r.client.QueryRow(ctx, q)
	err := row.Scan(&min_watch.ID, &min_watch.Uri, &min_watch.WatchCount)
	r.logger.Debugf("max err %v", err)
	return min_watch, err
}

func (r db) UpdateOne(ctx context.Context, video video_db_dto.Video) error {
	q := `UPDATE public.videos 
	SET 
	uri = $2,
	watch_count = $3
	WHERE id = $1;
	`
	_, err := r.client.Exec(ctx, q, &video.ID, &video.Uri, &video.WatchCount)
	return err
}
