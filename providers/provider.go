package providers

type TorrentResult struct {
	Title  string
	Magnet string
	Source string
	Size   string
	Age    string
	Seeds  string
	Peers  string
}

func PartialResult(title string, source string, size string, age string, seeds string, peers string) TorrentResult {
	return TorrentResult{
		Title:  title,
		Source: source,
		Size:   size,
		Age:    age,
		Seeds:  seeds,
		Peers:  peers,
	}
}

type TorrentProvider interface {
	Get(query string) []TorrentResult
}
