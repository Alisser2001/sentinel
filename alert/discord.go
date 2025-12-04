package alert

import (
	"bytes"
	"net/http"
)

func SendDiscord(webhookURL, msg string) error {
	if webhookURL == "" {
		return nil
	}

	body := []byte(`{"content":"` + msg + `"}`)
	_, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(body))
	return err
}