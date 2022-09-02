package video

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"
	"video_service/internal/controller/video_service/video_logic"
	"video_service/pkg/utils"

	"github.com/fasthttp/router"
	"github.com/pion/randutil"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/oggreader"
	"github.com/valyala/fasthttp"
)

const (
	oggPageDuration = time.Millisecond * 20
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

	// Create channel that is blocked until ICE Gathering is complete
	// gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	answer, err := Peer_pool[user_id].CreateAnswer(nil)
	if err != nil {
		panic(err)
	} else if err = Peer_pool[user_id].SetLocalDescription(answer); err != nil {
		panic(err)
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	// <-gatherComplete

	response, errMarshal := json.Marshal(*Peer_pool[user_id].LocalDescription())
	if err != nil {
		panic(errMarshal)
	}
	//log.Printf("LocalDescription RESPONSE \n ___________________ %v", *Peer_pool[user_id].LocalDescription())
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
			/*{
				URLs:       []string{"turn:TURN_IP:3478?transport=tcp"},
				Username:   "username",
				Credential: "password",
			},*/
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
			// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
			// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
			// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
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

	// ПОДМЕНИТЬ НА ВНЕШНИЙ ПОТОК С КАМЕРЫ
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

func writeAudioToTrack(audioTrack *webrtc.TrackLocalStaticSample, user_id string) {

	var file_audio *os.File
	var err error
	switch user_id {
	case "0":
		log.Print("start write 0")
		file_audio, err = os.Open("output.ogg")
		if err != nil {
			panic(err)
		}

	default:
		log.Printf("start write %v", fmt.Sprintf("output_%v.ogg", user_id))
		file_audio, err = os.Open(fmt.Sprintf("output_%v.ogg", user_id))
		if err != nil {
			log.Printf("error file is %v", file_audio)
			return
		}

	}
	log.Printf("file audio %v", file_audio.Name())
	// Open on oggfile in non-checksum mode.
	ogg, _, oggErr := oggreader.NewWith(file_audio)
	if oggErr != nil {
		log.Printf("Warning OggErr %v", oggErr)
		return
	}

	// Wait for connection established

	// Keep track of last granule, the difference is the amount of samples in the buffer
	var lastGranule uint64

	// It is important to use a time.Ticker instead of time.Sleep because
	// * avoids accumulating skew, just calling time.Sleep didn't compensate for the time spent parsing the data
	// * works around latency issues with Sleep (see https://github.com/golang/go/issues/44343)
	ticker := time.NewTicker(oggPageDuration)
	for ; true; <-ticker.C {
		if _, exists := Peer_pool[user_id]; !exists {
			return
		} else {
			if Peer_pool[user_id].ConnectionState().String() == "closed" {
				return
			}
		}

		pageData, pageHeader, oggErr := ogg.ParseNextPage()
		if errors.Is(oggErr, io.EOF) {
			fmt.Printf("All audio pages parsed and sent")
			return
		}

		if oggErr != nil {
			panic(oggErr)
		}

		// The amount of samples is the difference between the last and current timestamp
		sampleCount := float64(pageHeader.GranulePosition - lastGranule)
		lastGranule = pageHeader.GranulePosition
		sampleDuration := time.Duration((sampleCount/48000)*1000) * time.Millisecond
		// log.Printf("|user_id|%v|| write audio %v\n", user_id, sampleDuration)
		if oggErr = audioTrack.WriteSample(media.Sample{Data: pageData, Duration: sampleDuration}); oggErr != nil {
			panic(oggErr)
		}
	}

}

func (service *restClient) writeVideoToTrack(t *webrtc.TrackLocalStaticSample, user_id string) {
	defer service.video_service.Close()
	var err error
	//annexbNALUStartCode := func() []byte { return []byte{0x00, 0x00, 0x00, 0x01} }
	//session, err := rtsp.Dial(fmt.Sprintf("rtsp://admin:Windowsmac13@192.168.1.64:554/ISAPI/Streaming/Channels/%v", user_id))

	if err != nil {
		panic(err)
	}
	// session.RtpKeepAliveTimeout = 10 * time.Second
	// codecs, err := session.Streams()
	// if err != nil {
	// 	panic(err)
	// }
	// for i, t := range codecs {
	// 	log.Println("Stream", i, "is of type", t.Type().String())
	// }
	// if codecs[0].Type() != av.H264 {
	// 	panic("RTSP feed must begin with a H264 codec")
	// }

	// if err != nil {
	// 	panic(err)
	// }
	// log.Printf("ID %v| StreamID %v| t.Kind() %v|", t.ID(), t.StreamID(), t.Kind())
	var previousTime time.Duration
	new_service := service.video_service.AddTrack(fmt.Sprintf("rtsp://admin:Windowsmac13@192.168.1.64:554/ISAPI/Streaming/Channels/%v", user_id), 10*time.Second)

	for {
		if _, exists := Peer_pool[user_id]; !exists {
			break
		} else {
			if Peer_pool[user_id].ConnectionState().String() == "closed" {
				break
			}
		}
		log.Print("scan files")
		data, pkt_time := new_service.Read()

		// pkt, err := session.ReadPacket()
		// if err != nil {
		// 	break
		// }

		// if pkt.Idx != 0 {
		// 	//audio or other stream, skip it
		// 	continue
		// }

		// pkt.Data = pkt.Data[4:]
		// if pkt.IsKeyFrame {
		// 	pkt.Data = append(annexbNALUStartCode(), pkt.Data...)
		// 	pkt.Data = append(codecs[0].(h264parser.CodecData).PPS(), pkt.Data...)
		// 	pkt.Data = append(annexbNALUStartCode(), pkt.Data...)
		// 	pkt.Data = append(codecs[0].(h264parser.CodecData).SPS(), pkt.Data...)
		// 	pkt.Data = append(annexbNALUStartCode(), pkt.Data...)
		// }

		bufferDuration := pkt_time - previousTime
		previousTime = pkt_time
		time.Sleep(pkt_time)
		log.Printf("sending file %v", bufferDuration)
		if h264Err := t.WriteSample(media.Sample{Data: data, Duration: pkt_time}); h264Err != nil && h264Err != io.ErrClosedPipe {
			panic(h264Err)
		}
	}
}
