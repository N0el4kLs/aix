package openai

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/projectdiscovery/aix/internal"
	"github.com/sashabaranov/go-openai"
)

type Source struct {
	client  *openai.Client
	options internal.Options
	chatReq openai.ChatCompletionRequest
}

func NewSource(options *internal.Options) (*Source, error) {
	var (
		model string
	)
	client := openai.NewClient(options.OpenaiApiKey)
	if options.Gpt3 {
		model = openai.GPT3Dot5Turbo
	}
	if options.Gpt4 {
		// use turbo preview by default
		model = openai.GPT4TurboPreview
	}
	if options.Model != "" {
		model = options.Model
	}

	chatReq := openai.ChatCompletionRequest{
		Model:    model,
		Messages: []openai.ChatCompletionMessage{},
	}

	if len(options.System) != 0 {
		chatReq.Messages = append(chatReq.Messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: strings.Join(options.System, "\n"),
		})
	}

	if options.Prompt != "" {
		chatReq.Messages = append(chatReq.Messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: options.Prompt,
		})
	}

	if options.Temperature != 0 {
		chatReq.Temperature = options.Temperature
	}
	if options.TopP != 0 {
		chatReq.TopP = options.TopP
	}

	return &Source{
		client:  client,
		options: *options,
	}, nil
}

func (s Source) ListModels() (*internal.Result, error) {
	models, err := s.client.ListModels(context.Background())
	if err != nil {
		return &internal.Result{}, err
	}
	var buff bytes.Buffer
	for _, model := range models.Models {
		buff.WriteString(fmt.Sprintf("%s\n", model.ID))
	}

	result := &internal.Result{
		Timestamp: time.Now().String(),
		Model:     s.options.Model,
		Prompt:    s.options.Prompt,
	}
	if s.options.Stream {
		result.SetupStreaming()
		go func(res *internal.Result) {
			defer res.CloseCompletionStream()
			res.WriteCompletionStreamResponse(buff.String())
		}(result)
	} else {
		result.Completion = buff.String()
	}

	return result, nil
}

func (s Source) ChatGenerate() (*internal.Result, error) {
	if len(s.chatReq.Messages) == 0 {
		return &internal.Result{}, fmt.Errorf("no prompt provided")
	}
	result := &internal.Result{
		Timestamp: time.Now().String(),
		Model:     s.options.Model,
		Prompt:    s.options.Prompt,
	}
	chatGptResp, err := s.client.CreateChatCompletion(context.TODO(), s.chatReq)
	if err != nil {
		return &internal.Result{Error: err}, err
	}
	if len(chatGptResp.Choices) == 0 {
		return &internal.Result{}, fmt.Errorf("no data on response")
	}
	result.Completion = chatGptResp.Choices[0].Message.Content

	return result, nil
}

func (s Source) StreamChatGenerate() (*internal.Result, error) {
	if len(s.chatReq.Messages) == 0 {
		return &internal.Result{}, fmt.Errorf("no prompt provided")
	}
	result := &internal.Result{
		Timestamp: time.Now().String(),
		Model:     s.options.Model,
		Prompt:    s.options.Prompt,
	}
	result.SetupStreaming()
	go func(res *internal.Result) {
		defer res.CloseCompletionStream()
		s.chatReq.Stream = true
		stream, err := s.client.CreateChatCompletionStream(context.TODO(), s.chatReq)
		if err != nil {
			res.Error = err
			return
		}
		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				res.Error = err
				return
			}
			if len(response.Choices) == 0 {
				res.Error = fmt.Errorf("got empty response")
				return
			}
			res.WriteCompletionStreamResponse(response.Choices[0].Delta.Content)
		}
	}(result)
	return result, nil
}
