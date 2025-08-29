package anna

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/iosifache/annas-mcp/internal/gemini"
	"github.com/iosifache/annas-mcp/internal/logger"
	"github.com/iosifache/annas-mcp/internal/types"
	"go.uber.org/zap"
	"golang.org/x/net/html"
)

const (
	AnnasSearchEndpoint   = "https://annas-archive.org/search?q=%s"
	AnnasDownloadEndpoint = "https://annas-archive.org/dyn/api/fast_download.json?md5=%s&key=%s"
)

type fastDownloadResponse struct {
	DownloadURL string `json:"download_url"`
	Error       string `json:"error"`
}

func Search(query string) ([]types.Book, error) {
	l := logger.GetLogger()
	fullURL := fmt.Sprintf(AnnasSearchEndpoint, url.QueryEscape(query))
	l.Info("Visiting URL", zap.String("url", fullURL))

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		// If parsing fails, we can still try to send the whole body.
		l.Warn("Failed to parse HTML, passing whole body to Gemini", zap.Error(err))
		return gemini.ExtractBookInfo(string(body))
	}

	var listNode *html.Node
	var findNode func(*html.Node)
	findNode = func(n *html.Node) {
		if listNode != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, a := range n.Attr {
				if a.Key == "class" && strings.Contains(a.Val, "js-aarecord-list-outer") {
					listNode = n
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findNode(c)
		}
	}
	findNode(doc)

	if listNode == nil {
		l.Warn("Could not find book list in HTML response, passing whole body to Gemini")
		return gemini.ExtractBookInfo(string(body))
	}

	var buf bytes.Buffer
	if err := html.Render(&buf, listNode); err != nil {
		return nil, err
	}

	return gemini.ExtractBookInfo(buf.String())
}

func Download(b *types.Book, secretKey, folderPath string) error {
	apiURL := fmt.Sprintf(AnnasDownloadEndpoint, b.Hash, secretKey)

	resp, err := http.Get(apiURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var apiResp fastDownloadResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return err
	}
	if apiResp.DownloadURL == "" {
		if apiResp.Error != "" {
			return errors.New(apiResp.Error)
		}
		return errors.New("failed to get download URL")
	}

	downloadResp, err := http.Get(apiResp.DownloadURL)
	if err != nil {
		return err
	}
	defer downloadResp.Body.Close()

	if downloadResp.StatusCode != http.StatusOK {
		return errors.New("failed to download file")
	}

	filename := b.Title + "." + b.Format
	filename = strings.ReplaceAll(filename, "/", "_")
	filePath := filepath.Join(folderPath, filename)

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, downloadResp.Body)
	return err
}

func String(b *types.Book) string {
	return fmt.Sprintf("Title: %s\nAuthors: %s\nPublisher: %s\nLanguage: %s\nFormat: %s\nSize: %s\nURL: %s\nHash: %s",
		b.Title, b.Authors, b.Publisher, b.Language, b.Format, b.Size, b.URL, b.Hash)
}

func ToJSON(b *types.Book) (string, error) {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}
