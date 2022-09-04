package db_contracts

import (
	"context"
	"video_service/internal/controllers/database/video/video_db_dto"
)

type LogicVideoDb interface {
	FindOne(ctx context.Context, id string) (video_db_dto.Video, error)
	FindAll(ctx context.Context) (u []video_db_dto.Video, err error)
	Create(ctx context.Context, video *video_db_dto.Video) error
	Delete(ctx context.Context, id string) error
	UpdateOne(ctx context.Context, video video_db_dto.Video) error
	FindWithUri(ctx context.Context, uri string) (video_db_dto.Video, error)
	MinWatch(ctx context.Context) (video_db_dto.Video, error)
	MaxWatch(ctx context.Context) (video_db_dto.Video, error)
}
