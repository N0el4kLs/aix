package runner

import (
	"github.com/projectdiscovery/aix/internal"
	"github.com/projectdiscovery/aix/internal/source"
	"github.com/projectdiscovery/aix/internal/source/gemini"
	"github.com/projectdiscovery/aix/internal/source/openai"
	errorutil "github.com/projectdiscovery/utils/errors"
)

const (
	OPENAI = "openai"
	GEMINI = "gemini"
)

// ErrNoKey is returned when no key is provided
var ErrNoKey = errorutil.New("OPENAI_API_KEY is not configured / provided.")

// Runner contains the internal logic of the program
type Runner struct {
	options *internal.Options
}

// NewRunner instance
func NewRunner(options *internal.Options) (*Runner, error) {
	return &Runner{
		options: options,
	}, nil
}

// Run the instance
func (r *Runner) Run() (*internal.Result, error) {
	//return Generate(r.options)
	var llmSource source.LLMSource
	switch r.options.LLMSource {
	case GEMINI:
		geminiSource, err := gemini.NewSource(r.options)
		if err != nil {
			return &internal.Result{}, err
		}
		llmSource = geminiSource
	case OPENAI:
		openaiSource, err := openai.NewSource(r.options)
		if err != nil {
			return &internal.Result{}, err
		}
		llmSource = openaiSource
	}
	if r.options.ListModels {
		return llmSource.ListModels()
	}
	switch {
	case r.options.Stream:
		return llmSource.StreamChatGenerate()
	default:
		return llmSource.ChatGenerate()
	}
}
