package config

import (
	"sync"
	"time"
	"video_service/pkg/logging"

	"github.com/ilyakaznacheev/cleanenv"
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

type Config struct {
	IsDebug       bool          `yaml:"is_debug" env-required:"true"`
	Storage       StorageConfig `yaml:"storage"`
	Project_state string        `yaml:"project_state"`
	Listen        struct {
		Port string `yaml:"port" env-default:"8080"`
	} `yaml:"listen"`
}

type StorageConfig struct {
	Host       string `json:"host"`
	Port       string `json:"port"`
	Database   string `json:"database"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	MaxRetries int    `json:"retries"`
}

var instance *Config
var once sync.Once

func GetConfig() *Config {
	once.Do(func() {
		logger := logging.GetLogger()
		logger.Info("read application configuration")
		instance = &Config{}
		if err := cleanenv.ReadConfig("config.yml", instance); err != nil {
			help, _ := cleanenv.GetDescription(instance, nil)
			logger.Info(help)
			logger.Fatal(err)
		}
		logger.Debugf("MaxRetries %v", instance.Storage.MaxRetries)
	})

	instance.Storage.MaxRetries = 5

	return instance
}

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
