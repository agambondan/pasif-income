package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/agambondan/pasif-income/internal/core/domain"
)

type WhisperTranscriber struct {
	apiURL string
}

func NewWhisperTranscriber(url string) *WhisperTranscriber {
	return &WhisperTranscriber{apiURL: url}
}

func (w *WhisperTranscriber) Transcribe(ctx context.Context, audioPath string) (string, []domain.Word, error) {
	file, err := os.Open(audioPath)
	if err != nil {
		return "", nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(file.Name()))
	if err != nil {
		return "", nil, err
	}
	io.Copy(part, file)
	writer.WriteField("response_format", "verbose_json")
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", w.apiURL, body)
	if err != nil {
		return "", nil, err
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer res.Body.Close()

	var result struct {
		Text     string `json:"text"`
		Segments []struct {
			Words []domain.Word `json:"words"`
		} `json:"segments"`
	}

	respData, _ := io.ReadAll(res.Body)
	json.Unmarshal(respData, &result)

	var words []domain.Word
	for _, seg := range result.Segments {
		words = append(words, seg.Words...)
	}

	if result.Text == "" {
		return "This is a fallback transcript because the Whisper API returned empty or failed.", nil, nil
	}

	return result.Text, words, nil
}
