package hickvision

import (
	"time"

	"video_service/pkg/utils"

	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/codec/h264parser"
	"github.com/nareix/joy4/format/rtsp"
)

type hickVisionService struct {
	session             *rtsp.Client
	RtpKeepAliveTimeout time.Duration
	codecs              []av.CodecData
}

// rtsp://admin:Windowsmac13@192.168.1.64:554/ISAPI/Streaming/Channels/0"
func NewHickVisionService(uri string, keppAlive time.Duration) hickVisionService {
	session, err := rtsp.Dial(uri)
	utils.CatchErr(err)
	codecs, err := session.Streams()
	utils.CatchErr(err)
	return hickVisionService{session: session, RtpKeepAliveTimeout: keppAlive, codecs: codecs}
}

func (service hickVisionService) ReadPacket() ([]byte, time.Duration) {
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

func (service hickVisionService) Close() {
	service.session.Close()
}
