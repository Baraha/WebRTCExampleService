package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"video_service/internal/adapters/postgresql"
	"video_service/internal/api/video"
	"video_service/internal/app/config"
	"video_service/internal/controller/database/video_db_logic"
	videologic "video_service/internal/controller/video_service/video_logic"
	"video_service/pkg/logging"

	"github.com/fasthttp/router"
	"github.com/pion/webrtc/v3"
	"github.com/valyala/fasthttp"
)

var rout *router.Router

func fileServer() {
	http.Handle("/", http.FileServer(http.Dir(".")))
	fmt.Println("Open http://localhost:9001 to access this demo")
	panic(http.ListenAndServe(":9001", nil))
}

func Init() {
	cfg := config.GetConfig()

	postgreSQLClient, err := postgresql.NewPostgresClient(context.TODO(), postgresql.StorageConfig(cfg.Storage))
	if err != nil {
		log.Fatalf("%v", err)
	}
	db_video := video_db_logic.NewDBLogic(postgreSQLClient, logging.GetLogger())

	log.Printf("postgr client %v", postgreSQLClient)

	rest_client := video.NewRestClient(videologic.NewVideoService(db_video))

	var media = webrtc.MediaEngine{}

	if err := media.RegisterDefaultCodecs(); err != nil {
		panic(err)
	}

	rest_client.RtcApi = webrtc.NewAPI(webrtc.WithMediaEngine(&media))

	config.Init(config.GetConfig().Project_state)
	r := router.New()

	rout = r
	rest_client.Register(r)
}

var (
	corsAllowHeaders     = "authorization"
	corsAllowMethods     = "HEAD,GET,POST,PUT,DELETE,OPTIONS"
	corsAllowOrigin      = "*"
	corsAllowCredentials = "true"
)

func CORS(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		origin := string(ctx.Request.Header.Peek("origin"))
		//strings.Replace(origin, "4200", "5000", 1)
		corsAllowOrigin = origin
		corsAllowHeaders = string(ctx.Request.Header.Peek("Access-Control-Request-Headers"))
		ctx.Response.Header.Set("Access-Control-Allow-Credentials", corsAllowCredentials)
		ctx.Response.Header.Set("Access-Control-Allow-Headers", corsAllowHeaders)
		ctx.Response.Header.Set("Access-Control-Allow-Methods", corsAllowMethods)
		ctx.Response.Header.Set("Access-Control-Allow-Origin", corsAllowOrigin)

		next(ctx)
	}
}

func Start() {
	//go fileServer()
	video.Peer_pool = make(map[string]*webrtc.PeerConnection)
	log.Printf("server is starting on %v!", config.GetConfig().Listen.Port)
	if err := fasthttp.ListenAndServe(fmt.Sprintf(":%v", config.GetConfig().Listen.Port), CORS(rout.Handler)); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}
