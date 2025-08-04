package anna

import (
	"io/ioutil"
	"net/url"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseBooks(t *testing.T) {
	// Read the sample HTML file
	htmlContent, err := ioutil.ReadFile(filepath.Join("testdata", "hearnshaw_search.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	// Create a dummy URL for parsing
	pageURL, _ := url.Parse("")

	// Call the function to be tested
	books, err := parseBooks(string(htmlContent), pageURL)
	if err != nil {
		t.Fatalf("parseBooks returned an error: %v", err)
	}

	// Define the expected result
	expectedBooks := []*Book{
		{
			Language:  "English [en]",
			Format:    ".zip",
			Size:      "0.1MB",
			Title:     "The development of political ideas, by F.J.C. Hearnshaw ...",
			Publisher: "E. Benn, Limited, 1931., England, 1931",
			Authors:   "Hearnshaw, F. J. C. 1869-1946.",
			Hash:      "fc57224f94300bfba438a54500eaabeb",
		},
	}

	// Compare the actual result with the expected result
	if len(books) != len(expectedBooks) {
		t.Fatalf("expected %d books, but got %d", len(expectedBooks), len(books))
	}

	for i, book := range books {
		expectedBook := expectedBooks[i]
		// We ignore the URL field as requested
		book.URL = ""
		if !reflect.DeepEqual(book, expectedBook) {
			t.Errorf("book %d does not match expected value.\nExpected: %+v\nGot:      %+v", i, expectedBook, book)
		}
	}
}
