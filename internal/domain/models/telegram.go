package models

type BotUpdate struct {
	UpdateID int     `json:"update_id"`
	Message  Message `json:"message"`
}

type Message struct {
	MessageID int    `json:"message_id"`
	From      User   `json:"from"`
	Text      string `json:"text"`
	Contact   struct {
		PhoneNumber string `json:"phone_number"`
	} `json:"contact"`
	Chat struct {
		ID int64 `json:"id"`
	} `json:"chat"`
}

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
}
