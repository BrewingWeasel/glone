package gitservice

type DirStructure []FileValues

type FileValues struct {
	Path        string `json:"path"`
	URL         string `json:"url"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
}
