package video_logic

import (
	"log"
	"time"
	fileservice "video_service/internal/adapters/file_service"
	"video_service/internal/adapters/hickvision"
	"video_service/internal/app/config"
	videocontracts "video_service/internal/controller/video_service/video_contracts"
)

type VideoLogicContract interface {
	Read() ([]byte, time.Duration)
	Close()
	AddTrack(uri string, keep_alive time.Duration) videoService
}

type videoService struct {
	video_client videocontracts.VideoContract
}

func NewVideoService() videoService {
	return videoService{}
}

func (service videoService) AddTrack(uri string, keep_alive time.Duration) videoService {
	switch config.VideoService {
	case config.STATE_PROD:
		service.video_client = hickvision.NewHickVisionService(uri, keep_alive)
	case config.STATE_DEV:
		log.Printf("state dev")
		service.video_client = fileservice.NewFileService()
		log.Printf("service is %v", service.video_client)
	}
	return service
}

func (service videoService) Read() ([]byte, time.Duration) {
	log.Printf("service video client %v", service.video_client)
	return service.video_client.ReadPacket()
}

func (service videoService) Close() {
	service.video_client.Close()
}
