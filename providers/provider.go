package providers

type TorrentResult struct {
	Title  string
	Magnet string
	Size   string
}

func Partial(title string) TorrentResult {
	var result = TorrentResult{}
	result.Title = title
	return result
}

type TorrentProvider interface {
	Get(query string) []TorrentResult
}
