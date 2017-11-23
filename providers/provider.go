package providers

type TorrentResult struct {
	Title string
	Url   string
}

type TorrentProvider interface {
	Get(query string) []TorrentResult
}
