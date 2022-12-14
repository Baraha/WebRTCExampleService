package cameras

import (
	"time"

	"video_service/pkg/utils"

	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/codec/h264parser"
	"github.com/nareix/joy4/format/rtsp"
)

type camService struct {
	session             *rtsp.Client
	RtpKeepAliveTimeout time.Duration
	codecs              []av.CodecData
}

// rtsp://admin:Windowsmac13@192.168.1.64:554/ISAPI/Streaming/Channels/0"
func NewCamService(uri string, keppAlive time.Duration) (camService, error) {
	session, err := rtsp.Dial(uri)
	if err != nil {
		return camService{}, err
	}

	codecs, err := session.Streams()
	if err != nil {
		return camService{}, err
	}

	return camService{session: session, RtpKeepAliveTimeout: keppAlive, codecs: codecs}, nil
}

func (service camService) ReadPacket() ([]byte, time.Duration) {
	annexbNALUStartCode := func() []byte { return []byte{0x00, 0x00, 0x00, 0x01} }

	pkt, err := service.session.ReadPacket()
	utils.CatchErr(err)

	if pkt.Idx != 0 {
		//audio or other stream, skip it
		return nil, pkt.Time
	}

	pkt.Data = pkt.Data[4:]
	if pkt.IsKeyFrame {
		pkt.Data = append(annexbNALUStartCode(), pkt.Data...)
		pkt.Data = append(service.codecs[0].(h264parser.CodecData).PPS(), pkt.Data...)
		pkt.Data = append(annexbNALUStartCode(), pkt.Data...)
		pkt.Data = append(service.codecs[0].(h264parser.CodecData).SPS(), pkt.Data...)
		pkt.Data = append(annexbNALUStartCode(), pkt.Data...)
	}
	utils.CatchErr(err)
	return pkt.Data, pkt.Time
}

func (service camService) Close() {
	service.session.Close()
}
