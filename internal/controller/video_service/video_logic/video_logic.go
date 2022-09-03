package video_logic

import (
	"context"
	"log"
	"time"
	"video_service/internal/adapters/cameras"
	fileservice "video_service/internal/adapters/file_service"
	"video_service/internal/app/config"
	"video_service/internal/controller/database/db_contracts"
	"video_service/internal/controller/database/dto_video_db"
	videocontracts "video_service/internal/controller/video_service/video_contracts"
	"video_service/pkg/utils"

	"github.com/google/uuid"
)

type VideoLogicContract interface {
	Read() ([]byte, time.Duration)
	Close(id string)
	AddTrack(uri string, keep_alive time.Duration) (videoService, string)
	GetAllVideos() []dto_video_db.Video
}

type videoService struct {
	video_client videocontracts.VideoContract
	db_client    db_contracts.LogicVideoDb
}

func NewVideoService(db db_contracts.LogicVideoDb) videoService {
	return videoService{db_client: db}
}

func (service videoService) AddTrack(uri string, keep_alive time.Duration) (videoService, string) {
	switch config.VideoService {
	case config.STATE_PROD:
		service.video_client = cameras.NewCamService(uri, keep_alive)
	case config.STATE_DEV:
		log.Printf("state dev")
		service.video_client = fileservice.NewFileService()
		log.Printf("service is %v", service.video_client)
	}
	new_id := uuid.NewString()
	service.db_client.Create(context.TODO(), &dto_video_db.Video{
		Uri: uri,
		ID:  new_id,
	})

	return service, new_id
}

func (service videoService) Read() ([]byte, time.Duration) {
	return service.video_client.ReadPacket()
}

func (service videoService) Close(id string) {
	service.db_client.Delete(context.TODO(), id)
	service.video_client.Close()
}

func (service videoService) GetAllVideos() []dto_video_db.Video {
	dto, err := service.db_client.FindAll(context.TODO())
	utils.CatchErr(err)
	return dto
}
