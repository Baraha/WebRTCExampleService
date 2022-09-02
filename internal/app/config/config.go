package config

import (
	"time"
)

const (
	STATIC_CACHE_TIME time.Duration = 172800 * time.Second
	DB_NAME           string        = "test"
	STATE_DOCKER                    = "docker"
	STATE_DEV                       = "dev"
	STATE_PROD                      = "prod"
	STATE_SHELL                     = "shell"
	STATE_TEST                      = "test"
	DB_LABELS_LOGIC                 = 2
	REDIS_ADDR_DOCKER               = "redis:6379"
	REDIS_ADDR_SHELL                = "localhost:6379"
	INVALID_BODY_ERR                = "Invalid request body"
	SERVICE_PORT                    = 4200
	H264FrameDuration               = time.Millisecond * 33
)

var MONGO_URL string

var VideoService string

var STATE_REDIS string

var STATUS_SUCCESS map[string]interface{} = map[string]interface{}{"status": "success"}

var Mock_redis map[string]interface{}

/* Init : init project var */
func Init(conf_type string) {

	switch conf_type {
	case STATE_SHELL:

	case STATE_TEST:

	case STATE_PROD:
		VideoService = STATE_PROD

	case STATE_DEV:
		VideoService = STATE_DEV
	}

}
