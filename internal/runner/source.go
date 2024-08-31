package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/projectdiscovery/aix/internal/source/gemini"
	"github.com/sashabaranov/go-openai"
)

type Source interface {
	ListModels()
	ChatCompletion()
}

const (
	OPENAI                   = "openai"
	GEMINI                   = "gemini"
	ChatMessageRoleSystem    = "system"
	ChatMessageRoleUser      = "user"
	ChatMessageRoleAssistant = "assistant"
	ChatMessageRoleFunction  = "function"
	ChatMessageRoleTool      = "tool"
)

func Generate(options *Options) (*Result, error) {
	var (
		model          string
		generateResult *Result
	)
	if options.LLMSource == OPENAI {
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

		if options.ListModels {
			models, err := client.ListModels(context.Background())
			if err != nil {
				return &Result{}, err
			}
			var buff bytes.Buffer
			for _, model := range models.Models {
				buff.WriteString(fmt.Sprintf("%s\n", model.ID))
			}

			result := &Result{
				Timestamp: time.Now().String(),
				Model:     model,
				Prompt:    options.Prompt,
			}

			if options.Stream {
				result.SetupStreaming()
				go func(res *Result) {
					defer res.CloseCompletionStream()
					res.WriteCompletionStreamResponse(buff.String())
				}(result)
			} else {
				result.Completion = buff.String()
			}
			return result, nil
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

		if len(chatReq.Messages) == 0 {
			return &Result{}, fmt.Errorf("no prompt provided")
		}

		if options.Temperature != 0 {
			chatReq.Temperature = options.Temperature
		}
		if options.TopP != 0 {
			chatReq.TopP = options.TopP
		}

		result := &Result{
			Timestamp: time.Now().String(),
			Model:     model,
			Prompt:    options.Prompt,
		}
		switch {
		case options.Stream:
			// stream response
			result.SetupStreaming()
			go func(res *Result) {
				defer res.CloseCompletionStream()
				chatReq.Stream = true
				stream, err := client.CreateChatCompletionStream(context.TODO(), chatReq)
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
		default:
			chatGptResp, err := client.CreateChatCompletion(context.TODO(), chatReq)
			if err != nil {
				return &Result{Error: err}, err
			}
			if len(chatGptResp.Choices) == 0 {
				return &Result{}, fmt.Errorf("no data on response")
			}
			result.Completion = chatGptResp.Choices[0].Message.Content
		}

		return result, nil
	} else if options.LLMSource == GEMINI {
		client, err := gemini.GenGeminiClient()
		if err != nil {
			return &Result{}, err
		}
		// Todo default model, need to optimized later
		model = "gemini-1.5-flash"

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

		//if options.Prompt != "" {
		//	cs.History = append(cs.History,
		//		&genai.Content{
		//			Parts: []genai.Part{
		//				genai.Text(options.Prompt),
		//			},
		//			Role: ChatMessageRoleUser,
		//		})
		//}
		//if len(cs.History) == 0 {
		//	return &Result{}, fmt.Errorf("no prompt provided")
		//}

		// todo handle option
		if options.Temperature != 0 {

		}
		if options.TopP != 0 {

		}
		generateResult = &Result{
			Timestamp: time.Now().String(),
			Model:     model,
			Prompt:    options.Prompt,
		}

		switch {
		case options.Stream:
			generateResult.SetupStreaming()
		default:
			resp, err := cs.SendMessage(context.TODO(), genai.Text(options.Prompt))
			if err != nil {
				return &Result{Error: err}, err
			}
			if resp.Candidates[0].Content == nil {
				return &Result{}, fmt.Errorf("no data on response")
			}
			text := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
			generateResult.Completion = text
		}
	}

	return generateResult, nil
}
