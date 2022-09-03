package db_contracts

import (
	"context"
	"video_service/internal/controller/database/dto_video_db"
)

type LogicVideoDb interface {
	FindOne(ctx context.Context, id string) (dto_video_db.Video, error)
	FindAll(ctx context.Context) (u []dto_video_db.Video, err error)
	Create(ctx context.Context, video *dto_video_db.Video) error
	Delete(ctx context.Context, id string) error
	UpdateOne(ctx context.Context, video dto_video_db.Video) error
	FindWithUri(ctx context.Context, uri string) (dto_video_db.Video, error)
	MinWatch(ctx context.Context) int
	MaxWatch(ctx context.Context) int
}
