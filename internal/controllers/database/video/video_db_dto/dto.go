package video_db_dto

type Video struct {
	Uri        string `json:"uri"`
	ID         string `json:"id"`
	WatchCount int    `json:"watch_count"`
}

type CreateVideoDTO struct {
	ID  string `json:"id"`
	Uri string `json:"uri"`
}
