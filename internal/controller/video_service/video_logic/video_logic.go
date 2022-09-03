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

	"github.com/google/uuid"
)

type VideoLogicContract interface {
	Read() ([]byte, time.Duration)
	Close()
	AddTrack(uri string, keep_alive time.Duration) videoService
}

type videoService struct {
	video_client videocontracts.VideoContract
	db_client    db_contracts.LogicVideoDb
	id           string
}

func NewVideoService(db db_contracts.LogicVideoDb) videoService {
	return videoService{db_client: db}
}

func (service videoService) AddTrack(uri string, keep_alive time.Duration) videoService {
	switch config.VideoService {
	case config.STATE_PROD:
		service.video_client = cameras.NewCamService(uri, keep_alive)
	case config.STATE_DEV:
		log.Printf("state dev")
		service.video_client = fileservice.NewFileService()
		log.Printf("service is %v", service.video_client)
	}
	new_id := uuid.NewString()
	service.id = new_id
	service.db_client.Create(context.TODO(), &dto_video_db.Video{
		Uri: uri,
		ID:  new_id,
	})

	return service
}

func (service videoService) Read() ([]byte, time.Duration) {
	log.Printf("service video client %v", service.video_client)
	return service.video_client.ReadPacket()
}

func (service videoService) Close() {
	service.db_client.Delete(context.TODO(), service.id)
	service.video_client.Close()
}
