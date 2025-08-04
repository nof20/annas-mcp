package anna

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/iosifache/annas-mcp/internal/logger"
	"go.uber.org/zap"
)

const (
	AnnasSearchEndpoint   = "https://annas-archive.org/search?q=%s"
	AnnasDownloadEndpoint = "https://annas-archive.org/dyn/api/fast_download.json?md5=%s&key=%s"
)

func extractMetaInformation(meta string) (language, format, size string) {
	parts := strings.Split(meta, ", ")
	if len(parts) < 4 {
		return "", "", ""
	}

	language = parts[0]
	format = parts[1]
	size = parts[3]

	return language, format, size
}

func downloadHTML(query string) (string, *url.URL, error) {
	l := logger.GetLogger()
	fullURL := fmt.Sprintf(AnnasSearchEndpoint, url.QueryEscape(query))
	l.Info("Visiting URL", zap.String("url", fullURL))

	resp, err := http.Get(fullURL)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	return string(body), resp.Request.URL, nil
}

func parseBooks(htmlContent string, pageURL *url.URL) ([]*Book, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	bookListParsed := make([]*Book, 0)
	doc.Find("#aarecord-list > div > a[href^='/md5/']").Each(func(i int, s *goquery.Selection) {
		infoContainer := s.Find("div.relative.top-\\[-1\\].pl-4.grow.overflow-hidden")

		meta := infoContainer.Find("div").Eq(0).Text()
		title := infoContainer.Find("h3").Text()
		publisher := infoContainer.Find("div").Eq(1).Text()
		authors := infoContainer.Find("div").Eq(2).Text()

		language, format, size := extractMetaInformation(meta)

		link, _ := s.Attr("href")
		hash := strings.TrimPrefix(link, "/md5/")

		absoluteLink, err := pageURL.Parse(link)
		if err != nil {
			return
		}

		book := &Book{
			Language:  strings.TrimSpace(language),
			Format:    strings.TrimSpace(format),
			Size:      strings.TrimSpace(size),
			Title:     strings.TrimSpace(title),
			Publisher: strings.TrimSpace(publisher),
			Authors:   strings.TrimSpace(authors),
			URL:       absoluteLink.String(),
			Hash:      hash,
		}

		bookListParsed = append(bookListParsed, book)
	})

	return bookListParsed, nil
}

func FindBook(query string) ([]*Book, error) {
	htmlContent, pageURL, err := downloadHTML(query)
	if err != nil {
		return nil, err
	}
	return parseBooks(htmlContent, pageURL)
}

func (b *Book) Download(secretKey, folderPath string) error {
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

func (b *Book) String() string {
	return fmt.Sprintf("Title: %s\nAuthors: %s\nPublisher: %s\nLanguage: %s\nFormat: %s\nSize: %s\nURL: %s\nHash: %s",
		b.Title, b.Authors, b.Publisher, b.Language, b.Format, b.Size, b.URL, b.Hash)
}

func (b *Book) ToJSON() (string, error) {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}
