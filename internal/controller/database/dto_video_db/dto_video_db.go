package dto_video_db

type Video struct {
	Uri string `json:"uri"`
	ID  string `json:"id"`
}

type CreateVideoDTO struct {
	ID  string `json:"id"`
	Uri string `json:"uri"`
}
