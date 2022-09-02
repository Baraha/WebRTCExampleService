package video

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"
	"video_service/internal/controller/video_service/video_logic"
	"video_service/pkg/utils"

	"github.com/fasthttp/router"
	"github.com/pion/randutil"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/valyala/fasthttp"
)

var Peer_pool map[string]*webrtc.PeerConnection

type restClient struct {
	video_service video_logic.VideoLogicContract
	RtcApi        *webrtc.API
}

func NewRestClient(v video_logic.VideoLogicContract) restClient {
	return restClient{video_service: v}
}

func (service *restClient) Register(r *router.Router) {
	r.POST("/stream/start/", service.START_STREAM)
	r.POST("/stream/close/", service.CLOSE_STREAM)
	r.POST("/stream/init/", service.INIT_PEER)
}

func doSignaling(ctx *fasthttp.RequestCtx) {
	//log.Printf("REMOTE IP :%v", ctx.RemoteIP().String())
	user_id := string(string(ctx.QueryArgs().Peek("id")))
	if user_id == "" {
		ctx.Response.Header.Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]interface{}{"Error": "invalid user"})
		ctx.Response.AppendBody(b)
	}
	var offer webrtc.SessionDescription

	if err := json.Unmarshal(ctx.Request.Body(), &offer); err != nil {
		panic(err)
	}

	if err := Peer_pool[user_id].SetRemoteDescription(offer); err != nil {
		panic(err)
	}

	answer, err := Peer_pool[user_id].CreateAnswer(nil)
	if err != nil {
		panic(err)
	} else if err = Peer_pool[user_id].SetLocalDescription(answer); err != nil {
		panic(err)
	}

	response, errMarshal := json.Marshal(*Peer_pool[user_id].LocalDescription())
	if err != nil {
		panic(errMarshal)
	}
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.Response.AppendBody(response)
}

func (service *restClient) INIT_PEER(ctx *fasthttp.RequestCtx) {
	user_id := string(string(ctx.QueryArgs().Peek("id")))
	if user_id == "" {
		ctx.Response.Header.Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]interface{}{"Error": "invalid user"})
		ctx.Response.AppendBody(b)
	}
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback,
	}
	peerConnection, err := service.RtcApi.NewPeerConnection(config)

	if err != nil {
		panic(err)
	}
	state := peerConnection.ConnectionState()
	if state != webrtc.PeerConnectionStateNew {
		panic(fmt.Sprintf("createPeerConnection called in non-new state (%s)", peerConnection.ConnectionState()))
	}
	Peer_pool[user_id] = peerConnection

	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection State has changed: %s\n", s.String())

		if s == webrtc.PeerConnectionStateFailed {

			fmt.Println("Peer Connection has gone to failed exiting")
		}
	})
	doSignaling(ctx)
	fmt.Println("PeerConnection has been created")
}

func (service *restClient) START_STREAM(ctx *fasthttp.RequestCtx) {
	log.Printf("start streaming")
	user_id := string(string(ctx.QueryArgs().Peek("id")))
	if user_id == "" {
		ctx.Response.Header.Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]interface{}{"Error": "invalid user"})
		ctx.Response.AppendBody(b)
	}

	if _, exists := Peer_pool[user_id]; !exists {
		ctx.Response.Header.Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]interface{}{"Error": "user is nil"})
		ctx.Response.AppendBody(b)
		return
	}

	if Peer_pool[user_id].ConnectionState().String() == "closed" {
		ctx.Response.Header.Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]interface{}{"Error": "user conn was closed"})
		ctx.Response.AppendBody(b)
		return
	}

	videoTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264},
		fmt.Sprintf("video-%d", randutil.NewMathRandomGenerator().Uint32()),
		fmt.Sprintf("video-%d", randutil.NewMathRandomGenerator().Uint32()),
	)
	if err != nil {
		panic(err)
	}

	rtpSender_video, err := Peer_pool[user_id].AddTrack(videoTrack)
	if err != nil {
		panic(err)
	}

	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			_, attr, rtcpErr := rtpSender_video.Read(rtcpBuf)
			log.Printf("attr %v", attr)
			if rtcpErr != nil {
				return
			}

		}
	}()

	go service.writeVideoToTrack(videoTrack, user_id)
	doSignaling(ctx)
}
func (service *restClient) CLOSE_STREAM(ctx *fasthttp.RequestCtx) {

	user_id := string(string(ctx.QueryArgs().Peek("id")))
	if user_id == "" {
		ctx.Response.Header.Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]interface{}{"Error": "invalid user"})
		ctx.Response.AppendBody(b)
	}
	if _, exists := Peer_pool[user_id]; !exists {
		ctx.Response.Header.Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]interface{}{"Error": "user is nil"})
		ctx.Response.AppendBody(b)
		return
	}

	if Peer_pool[user_id].ConnectionState().String() == "closed" {
		ctx.Response.Header.Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]interface{}{"Error": "user conn was closed"})
		ctx.Response.AppendBody(b)
		return
	}

	if senders := Peer_pool[user_id].GetSenders(); len(senders) != 0 {
		if err := Peer_pool[user_id].RemoveTrack(senders[0]); err != nil {
			panic(err)
		}
	}
	doSignaling(ctx)
	utils.CatchErr(Peer_pool[user_id].Close())

}

func (service *restClient) writeVideoToTrack(t *webrtc.TrackLocalStaticSample, user_id string) {
	defer service.video_service.Close()
	var err error

	if err != nil {
		panic(err)
	}

	var previousTime time.Duration

	// Производим подключение к камере -  на dev среде стоит обработка из файла
	new_service := service.video_service.AddTrack(fmt.Sprintf("rtsp://admin:Windowsmac13@192.168.1.64:554/ISAPI/Streaming/Channels/%v", user_id), 10*time.Second)

	for {
		// Проверяем что peer все еше присутствует
		if _, exists := Peer_pool[user_id]; !exists {
			break
		} else {
			if Peer_pool[user_id].ConnectionState().String() == "closed" {
				break
			}
		}
		log.Print("scan files")
		data, pkt_time := new_service.Read()
		bufferDuration := pkt_time - previousTime
		previousTime = pkt_time
		time.Sleep(pkt_time)
		log.Printf("sending file %v", bufferDuration)
		if h264Err := t.WriteSample(media.Sample{Data: data, Duration: pkt_time}); h264Err != nil && h264Err != io.ErrClosedPipe {
			panic(h264Err)
		}
	}
}
