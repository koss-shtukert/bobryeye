package telegram

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
)

type Client struct {
	token  string
	chatID int64
	log    zerolog.Logger
}

func New(token string, chatID int64, log zerolog.Logger) *Client {
	return &Client{token: token, chatID: chatID, log: log}
}

func (c *Client) SendPhoto(path string, caption string) error {
	file, err := os.Open(path)
	if err != nil {
		c.log.Error().Err(err).Str("file", path).Msg("failed to open photo")
		return err
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("photo", filepath.Base(path))
	if err != nil {
		return err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}

	_ = writer.WriteField("chat_id", fmt.Sprintf("%d", c.chatID))
	_ = writer.WriteField("caption", caption)
	writer.Close()

	url := "https://api.telegram.org/bot" + c.token + "/sendPhoto"
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.log.Error().Err(err).Msg("failed to send photo to Telegram")
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.log.Error().Int("status", resp.StatusCode).Msg("telegram response error")
	}

	return nil
}
