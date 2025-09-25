package messages

type EmailMessage struct {
	Username    string  `json:"from"`
	Password    string  `json:"password"`
	NumMessages *uint32 `json:"num_messages"`
}

const (
	TypeEmailNotify = "email:notify"
)
