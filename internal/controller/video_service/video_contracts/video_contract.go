package videocontracts

import "time"

type VideoContract interface {
	ReadPacket() ([]byte, time.Duration)
	Close()
}
