package source

import (
	"github.com/projectdiscovery/aix/internal"
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
