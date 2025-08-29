package types

type Book struct {
	Language  string `json:"language"`
	Format    string `json:"format"`
	Size      string `json:"size"`
	Title     string `json:"title"`
	Publisher string `json:"publisher"`
	Authors   string `json:"authors"`
	URL       string `json:"url"`
	Hash      string `json:"hash"`
}
