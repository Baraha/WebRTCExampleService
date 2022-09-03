package video

import (
	"fmt"
	"io"
	"log"
	"time"
	"video_service/internal/controller/video_service/video_logic"

	"github.com/fasthttp/router"
	jsoniter "github.com/json-iterator/go"
	"github.com/pion/randutil"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/valyala/fasthttp"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary
var Peer_pool map[string]*webrtc.PeerConnection

type restClient struct {
	video_service video_logic.VideoLogicContract
	RtcApi        *webrtc.API
}

func NewRestClient(v video_logic.VideoLogicContract) restClient {
	return restClient{video_service: v}
}

func (service *restClient) Register(r *router.Router) {
	r.POST("/stream/start/", service.StartStream)
	r.POST("/stream/close/", service.CloseStream)
	r.POST("/stream/init/", service.InitPeer)
	r.GET("/connections/", service.GetVideo)
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
		log.Printf("panic at err 1")
		panic(err)
	}

	answer, err := Peer_pool[user_id].CreateAnswer(nil)
	if err != nil {
		log.Printf("panic at err 2")
		panic(err)
	} else if err = Peer_pool[user_id].SetLocalDescription(answer); err != nil {
		log.Printf("panic at err 3")
		panic(err)
	}

	response, errMarshal := json.Marshal(*Peer_pool[user_id].LocalDescription())
	if err != nil {
		log.Printf("panic at err 4")
		panic(errMarshal)
	}
	log.Printf("set resp")
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.Response.AppendBody(response)
}

func (service *restClient) InitPeer(ctx *fasthttp.RequestCtx) {
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

func (service *restClient) StartStream(ctx *fasthttp.RequestCtx) {
	log.Printf("start streaming")
	user_id := string(string(ctx.QueryArgs().Peek("id")))
	if user_id == "" {
		ctx.Response.Header.Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]interface{}{"Error": "invalid user"})
		ctx.Response.AppendBody(b)
	}

	video_url := string(string(ctx.QueryArgs().Peek("video_url")))
	if user_id == "" {
		ctx.Response.Header.Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]interface{}{"Error": "invalid video_url"})
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

	go service.writeVideoToTrack(videoTrack, user_id, video_url)
	doSignaling(ctx)
}
func (service *restClient) CloseStream(ctx *fasthttp.RequestCtx) {

	user_id := string(string(ctx.QueryArgs().Peek("id")))
	if user_id == "" {
		ctx.Response.Header.Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]interface{}{"Error": "invalid user"})
		ctx.Response.AppendBody(b)
	}

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
	defer Peer_pool[user_id].Close()

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
	log.Print("CLOSE STREAM")

	doSignaling(ctx)
	log.Print("return in close_stream")
}

func (service *restClient) writeVideoToTrack(t *webrtc.TrackLocalStaticSample, user_id string, video_url string) {
	var err error

	if err != nil {
		panic(err)
	}

	// Производим подключение к камере -  на dev среде стоит обработка из файла
	new_service, video_id := service.video_service.AddTrack(video_url, 10*time.Second)

	for {
		// Проверяем что peer все еше присутствует
		if _, exists := Peer_pool[user_id]; exists {
			if Peer_pool[user_id].ConnectionState().String() == "closed" {
				Peer_pool[user_id] = nil
				break
			}
		}
		// читаем фрейм
		data, pkt_time := new_service.Read()
		time.Sleep(pkt_time)
		if data == nil {
			break
		}

		if h264Err := t.WriteSample(media.Sample{Data: data, Duration: pkt_time}); h264Err != nil && h264Err != io.ErrClosedPipe {
			log.Print("panic h264Err")
			panic(h264Err)
		}
	}
	new_service.Close(video_id)
}

func (service *restClient) GetVideo(ctx *fasthttp.RequestCtx) {
	b, _ := json.Marshal(service.video_service.GetAllVideos())
	ctx.Response.AppendBody(b)
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.Response.SetStatusCode(200)
}
