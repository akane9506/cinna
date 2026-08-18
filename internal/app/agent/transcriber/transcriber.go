package transcriber

import (
	"context"
	"fmt"
	"os"

	"github.com/akane9506/cinna/internal/app"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type VoiceTranscriber struct {
	client *openai.Client
}

func NewTranscriber(config *app.Config) *VoiceTranscriber {
	apiKey := config.OpenaiAPIKey
	openAIClient := openai.NewClient(
		option.WithAPIKey(apiKey),
	)
	return &VoiceTranscriber{
		client: &openAIClient,
	}
}

func (t *VoiceTranscriber) Transcribe(ctx context.Context, file *os.File) (string, error) {
	transcription, err := t.client.Audio.Transcriptions.New(
		ctx,
		openai.AudioTranscriptionNewParams{
			File:     file,
			Keywords: []string{"Cinna"},
			Model:    "gpt-transcribe",
			// can also add prompt here to increase the transcription quality in the future
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to get transcription: %w", err)
	}
	return transcription.Text, nil
}
