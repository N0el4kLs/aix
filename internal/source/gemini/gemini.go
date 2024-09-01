package gemini

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/N0el4kLs/aix/internal"
	"github.com/google/generative-ai-go/genai"
)

const ChatMessageRoleSystem = "system"

type Source struct {
	client      *genai.Client
	options     internal.Options
	chatSession *genai.ChatSession
}

func NewSource(options *internal.Options) (*Source, error) {
	client, err := GenGeminiClient()
	if err != nil {
		return &Source{}, err
	}

	model := "gemini-1.5-flash"
	options.Model = model

	geminiModel := client.GenerativeModel(model)
	cs := geminiModel.StartChat()

	if len(options.System) != 0 {
		content := strings.Join(options.System, "\n")
		cs.History = append(cs.History,
			&genai.Content{
				Parts: []genai.Part{
					genai.Text(content),
				},
				Role: ChatMessageRoleSystem,
			})
	}

	if options.Temperature != 0 {
		geminiModel.Temperature = &options.Temperature
	}
	if options.TopP != 0 {
		geminiModel.Temperature = &options.TopP
	}

	return &Source{
		client:      client,
		chatSession: cs,
		options:     *options,
	}, nil
}

func (s *Source) ListModels() (*internal.Result, error) {
	// Todo implement
	models := s.client.ListModels(context.TODO())
	var buff bytes.Buffer
	for {
		m, err := models.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			panic(err)
		}
		buff.WriteString(fmt.Sprintf("%s\n", strings.TrimSpace(m.Name)))
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

func (s *Source) ChatGenerate() (*internal.Result, error) {
	result := &internal.Result{
		Timestamp: time.Now().String(),
		Model:     s.options.Model,
		Prompt:    s.options.Prompt,
	}

	resp, err := s.chatSession.SendMessage(context.TODO(), genai.Text(s.options.Prompt))
	if err != nil {
		return &internal.Result{Error: err}, err
	}
	if resp.Candidates[0].Content == nil {
		return &internal.Result{}, fmt.Errorf("no data on response")
	}
	result.Completion = fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	return result, nil
}

func (s *Source) StreamChatGenerate() (*internal.Result, error) {
	result := &internal.Result{
		Timestamp: time.Now().String(),
		Model:     s.options.Model,
		Prompt:    s.options.Prompt,
	}
	result.SetupStreaming()
	go func(res *internal.Result) {
		defer res.CloseCompletionStream()
		iter := s.chatSession.SendMessageStream(context.TODO(), genai.Text(s.options.Prompt))
		for {
			resp, err := iter.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				res.Error = err
				return
			}
			if resp.Candidates[0].Content == nil {
				res.Error = fmt.Errorf("got empty response")
				return
			}

			res.WriteCompletionStreamResponse(fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]))
		}
	}(result)

	return result, nil
}

func GenGeminiClient() (*genai.Client, error) {
	var (
		client *genai.Client
		err    error
	)

	// https://github.com/google/generative-ai-go/pull/101/files#diff-8fbd919ed011e2c50e66043f0a337fe2ac487636628539eee55f51a15053a4a5
	if os.Getenv("Gemini_PROXY") != "" {
		c := &http.Client{Transport: &ProxyRoundTripper{
			APIKey:   os.Getenv("Gemini_API_KEY"),
			ProxyURL: os.Getenv("Gemini_PROXY"),
		}}
		client, err = genai.NewClient(context.Background(),
			option.WithHTTPClient(c),
			option.WithAPIKey(os.Getenv("Gemini_API_KEY")))
	} else {
		client, err = genai.NewClient(context.Background(),
			option.WithAPIKey(os.Getenv("Gemini_API_KEY")))
	}

	return client, err
}
