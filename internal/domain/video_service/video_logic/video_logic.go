package video_logic

import (
	"context"
	"log"
	"time"
	"video_service/internal/adapters/cameras"
	fileservice "video_service/internal/adapters/file_service"
	"video_service/internal/app/config"
	"video_service/internal/controllers/database/db_contracts"
	"video_service/internal/controllers/database/video/video_db_dto"
	videocontracts "video_service/internal/domain/video_service/video_contracts"
	"video_service/pkg/utils"

	"github.com/google/uuid"
)

type VideoLogicContract interface {
	Read() ([]byte, time.Duration)
	Close(id string)
	AddTrack(uri string, keep_alive time.Duration) (videoService, string, error)
	GetAllVideos() []video_db_dto.Video
	GetMaxWatched() video_db_dto.Video
	GetMinWatched() video_db_dto.Video
}

type videoService struct {
	video_client videocontracts.VideoContract
	db_client    db_contracts.LogicVideoDb
}

func NewVideoService(db db_contracts.LogicVideoDb) videoService {
	return videoService{db_client: db}
}

func (service videoService) AddTrack(uri string, keep_alive time.Duration) (videoService, string, error) {
	var video_id string
	var err error

	switch config.VideoService {
	case config.STATE_PROD:
		service.video_client, err = cameras.NewCamService(uri, keep_alive)
		if err != nil {
			return service, video_id, err
		}

	case config.STATE_DEV:
		log.Printf("state dev")
		service.video_client, err = fileservice.NewFileService(uri)
		log.Printf("service is %v", service.video_client)
		if err != nil {
			return service, video_id, err
		}

	}
	video, err := service.db_client.FindWithUri(context.TODO(), uri)
	if err != nil {
		if err.Error() == "no rows in result set" {
			log.Printf("error in findOne %v", err)
			video_id = uuid.NewString()
			service.db_client.Create(context.TODO(), &video_db_dto.Video{
				Uri:        uri,
				ID:         video_id,
				WatchCount: 1,
			})
		} else {
			return service, video_id, err
		}

	} else {
		video_id = video.ID
		video.WatchCount++
		service.db_client.UpdateOne(context.Background(), video)
	}

	return service, video_id, nil
}

func (service videoService) Read() ([]byte, time.Duration) {
	return service.video_client.ReadPacket()
}

func (service videoService) Close(id string) {
	log.Printf("id is %v", id)
	video, _ := service.db_client.FindOne(context.TODO(), id)
	if video.WatchCount <= 1 {
		service.db_client.Delete(context.TODO(), id)
		return
	}
	video.WatchCount--
	utils.CatchErr(service.db_client.UpdateOne(context.Background(), video))
	service.video_client.Close()
}

func (service videoService) GetAllVideos() []video_db_dto.Video {
	dto, err := service.db_client.FindAll(context.TODO())
	utils.CatchErr(err)
	return dto
}

func (service videoService) GetMaxWatched() video_db_dto.Video {
	video, err := service.db_client.MaxWatch(context.TODO())
	utils.CatchErr(err)
	return video
}

func (service videoService) GetMinWatched() video_db_dto.Video {
	video, err := service.db_client.MinWatch(context.TODO())
	utils.CatchErr(err)
	return video
}
