package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/iosifache/annas-mcp/internal/types"
	"google.golang.org/api/option"
)

func ExtractBookInfo(htmlContent string) ([]types.Book, error) {

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.5-flash-lite")
	model.ResponseMIMEType = "application/json"
	model.ResponseSchema = &genai.Schema{
		Type: genai.TypeArray,
		Items: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"language":  {Type: genai.TypeString, Description: "The language of the book"},
				"format":    {Type: genai.TypeString, Description: "The file format of the book (e.g., PDF, EPUB)"},
				"size":      {Type: genai.TypeString, Description: "The file size of the book"},
				"title":     {Type: genai.TypeString, Description: "The title of the book"},
				"publisher": {Type: genai.TypeString, Description: "The publisher of the book"},
				"authors":   {Type: genai.TypeString, Description: "The authors of the book, comma-separated"},
				"url":       {Type: genai.TypeString, Description: "The URL to the book's page on Anna's Archive"},
				"hash":      {Type: genai.TypeString, Description: "The MD5 hash from the download link"},
			},
			Required: []string{"language", "format", "size", "title", "publisher", "authors", "url", "hash"},
		},
	}

	prompt := fmt.Sprintf(`
		Please extract the list of matched book information from the following HTML content.  Ignore any partial matches.

		For each book, provide the following details:
		- Language
		- Format
		- Size
		- Title
		- Publisher
		- Authors
		- URL
		- Hash (from the download link)

		If no books are found, return an empty JSON array: []

		Here is the HTML content:
		%s
	`, htmlContent)

	var resp *genai.GenerateContentResponse
	const maxRetries = 5
	const initialBackoff = 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		resp, err = model.GenerateContent(ctx, genai.Text(prompt))
		if err == nil {
			break
		}

		if i < maxRetries-1 {
			backoff := initialBackoff * time.Duration(math.Pow(2, float64(i)))
			// TODO: Use a proper logger
			fmt.Printf("Gemini API call failed. Retrying in %v...\n", backoff)
			time.Sleep(backoff)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate content from Gemini after %d retries: %w", maxRetries, err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		// This can happen if Gemini finds no books, which is a valid case.
		return []types.Book{}, nil
	}

	part := resp.Candidates[0].Content.Parts[0]
	textPart, ok := part.(genai.Text)
	if !ok {
		return nil, fmt.Errorf("response part is not text")
	}

	var books []types.Book
	if err = json.Unmarshal([]byte(textPart), &books); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response from Gemini: %w", err)
	}

	return books, nil
}
