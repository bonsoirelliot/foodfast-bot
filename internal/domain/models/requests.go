package models

type SendMessageRequest struct {
	UserID int64  `json:"user_id"`
	Text   string `json:"text"`
}

type BotRequest struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type UserExistsRequest struct {
	UserID int64 `json:"user_id"`
}

type UserSingUpRequest struct {
	UserID int64  `json:"user_id"`
	Phone  string `json:"phone"`
	Name   string `json:"name"`
}
