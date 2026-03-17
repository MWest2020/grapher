package fixer

import (
	"context"
	"fmt"
	"os"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/gongoeloe/grapher/internal/analyzer"
)

// Fixer calls the Claude API to generate fix proposals for findings.
type Fixer struct {
	client anthropic.Client
}

// New creates a Fixer. Returns an error if ANTHROPIC_API_KEY is not set.
func New() (*Fixer, error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is not set")
	}
	client := anthropic.NewClient(option.WithAPIKey(key))
	return &Fixer{client: client}, nil
}

// Propose calls Claude with the finding's FixPrompt and returns the proposal text.
func (f *Fixer) Propose(ctx context.Context, finding analyzer.Finding) (string, error) {
	msg, err := f.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_20250514,
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(finding.FixPrompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("claude API call: %w", err)
	}
	if len(msg.Content) == 0 {
		return "", fmt.Errorf("empty response from Claude")
	}
	for _, block := range msg.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}
	return "", fmt.Errorf("no text block in Claude response")
}
