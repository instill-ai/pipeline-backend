package model

type Error struct {
	Status int32 `json:"status,omitempty"`

	Title string `json:"title,omitempty"`

	Detail string `json:"detail,omitempty"`
}
