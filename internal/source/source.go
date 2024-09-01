package source

import (
	"github.com/N0el4kLs/aix/internal"
)

const (
	OPENAI = "openai"
	GEMINI = "gemini"
)

type LLMSource interface {
	ChatGenerate() (*internal.Result, error)
	ListModels() (*internal.Result, error)
	StreamChatGenerate() (*internal.Result, error)
}

type LLMOptions struct {
}
