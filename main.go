package main

import (
	"github.com/GaruGaru/magnete/providers"
	"fmt"
)

func main() {

	var torrentz providers.TorrentProvider = providers.NewTorrentz("https://torrentz2.eu")

	var torrents = torrentz.Get("Siege")

	for i, torrent := range torrents {
		fmt.Printf("%d - %s: %s\n", i, torrent.Title, torrent.Url)
	}

}
