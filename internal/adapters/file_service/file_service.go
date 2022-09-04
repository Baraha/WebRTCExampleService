package fileservice

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"
	"video_service/internal/app/config"
	"video_service/pkg/utils"

	"github.com/pion/webrtc/v3/pkg/media/h264reader"
)

type H264fileService struct {
	file       *os.File
	h264Reader *h264reader.H264Reader
}

func NewFileService(file_name string) (H264fileService, error) {
	file, h264Err := os.Open(fmt.Sprintf("./videos/%v.h264", file_name))
	if h264Err != nil {
		return H264fileService{}, h264Err
	}

	reader, h264Err := h264reader.NewReader(file)
	if h264Err != nil {
		return H264fileService{}, h264Err
	}
	return H264fileService{file: file, h264Reader: reader}, nil
}

func (service H264fileService) ReadPacket() ([]byte, time.Duration) {
	// pkt, h264Err := in.ReadPacket()
	pkt, h264Err := service.h264Reader.NextNAL()
	if h264Err == io.EOF {
		log.Printf("All video frames parsed and sent")
		return nil, config.H264FrameDuration
	}
	utils.CatchErr(h264Err)

	return pkt.Data, config.H264FrameDuration
}

func (service H264fileService) Close() {
	log.Print("close file connect")
	service.file.Close()
}
