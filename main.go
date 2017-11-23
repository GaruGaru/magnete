package main

import (
	"github.com/GaruGaru/magnete/providers"
	"fmt"
	"flag"
)

func main() {

	var query = flag.String("query", "", "Search query")

	if *query != "" {
		var torrentz providers.TorrentProvider = providers.NewTorrentz("https://torrentz2.eu")
		var torrents = torrentz.Get(*query)

		for i, torrent := range torrents {
			fmt.Printf("%d - %s: %s\n", i, torrent.Title, torrent.Url)
		}
	}else{
		panic("Empty query.")
	}

}
