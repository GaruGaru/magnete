package providers

import (
	"github.com/yhat/scrape"
	"net/http"
	"golang.org/x/net/html/atom"
	"strings"
	"golang.org/x/net/html"
	"fmt"
	"net/url"
	"sync"
	"time"
	"net"
)

type Torrentz struct {
	url string
}

func NewTorrentz(url string) Torrentz {
	return Torrentz{url}
}

func (t Torrentz) Get(query string) []TorrentResult {
	var searchUrl = fmt.Sprintf("%s/search?f=%s", t.url, url.QueryEscape(query))
	return t.torrentList(searchUrl, torrentListMatcher)
}

func getRoot(url string) (*html.Node, error) {

	var transport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	var httpClient = &http.Client{
		Timeout:   time.Second * 5,
		Transport: transport,
	}

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}

	root, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	return root, nil
}

func (t Torrentz) torrentList(url string, matcher scrape.Matcher) []TorrentResult {
	var root, err = getRoot(url)
	var results []TorrentResult
	if err == nil {

		torrents := scrape.FindAll(root, matcher)

		resultsChannel := make(chan TorrentResult, 1000)
		var wg sync.WaitGroup
		wg.Add(len(torrents))
		//fmt.Printf("Scraping %d torrents\n", len(torrents))
		for _, torrentItem := range torrents {
			var title = scrape.Text(torrentItem)
			var itemUrl = scrape.Attr(torrentItem, "href")
			go t.scrapeItem(title, itemUrl, resultsChannel, &wg)
		}


		wg.Wait()
		close(resultsChannel)

		for result := range resultsChannel {
			results = append(results, result)
		}

	} else {
		panic(fmt.Sprintf("Unable to fetch results list for %s", url))
	}
	return results
}

func (t Torrentz) scrapeItem(title string, url string, results chan TorrentResult, wg *sync.WaitGroup) {
	defer wg.Done()
	 //fmt.Printf("Done scraping %s\n", title)

	var magnets, err = magnetList(fmt.Sprintf("%s%s", t.url, url))
	if err == nil {
		//fmt.Printf("Got %d magnets provider for %s\n", len(magnets), title)
		for _, m := range magnets {
			//fmt.Printf("Trying magnet provider %s\n", m)
			var magnetUrl, err = getMagnent(m)
			if err == nil {
				//fmt.Printf("Found magnet from %s\n", m)
				results <- TorrentResult{title, magnetUrl}
				break
			}
		}
	} else {
		//fmt.Printf("No magnets provider for %s\n", title)
	}

}

func getMagnent(url string) (string, error) {
	var root, err = getRoot(url)
	if err != nil {
		return "", err
	}
	var urls = scrape.FindAll(root, magnetUrlMatcher)
	if len(urls) > 0 {
		return scrape.Attr(urls[0], "href"), nil
	} else {
		return "no_magnet", fmt.Errorf("no magnet found in %s", url)
	}
}

func magnetList(url string) ([]string, error) {
	var root, err = getRoot(url)
	if err != nil {
		return nil, err
	}
	torrents := scrape.FindAll(root, magnetListMatcher)
	var results []string
	for _, article := range torrents {
		results = append(results, scrape.Attr(article, "href"))
	}
	return results, nil
}

func magnetUrlMatcher(n *html.Node) bool {
	if n.DataAtom == atom.A {
		return strings.HasPrefix(scrape.Attr(n, "href"), "magnet:")
	} else {
		return false
	}
}

func magnetListMatcher(n *html.Node) bool {
	if n.DataAtom == atom.A && scrape.Attr(n.Parent.Parent.Parent, "class") == "downlinks" {
		return scrape.Attr(n, "target") == "_blank"
	} else {
		return false
	}
}

func torrentListMatcher(n *html.Node) bool {
	if n.DataAtom == atom.A && scrape.Attr(n.Parent.Parent.Parent, "class") == "results" {
		value := scrape.Attr(n, "href")
		take := strings.HasPrefix(value, "/")
		return take
	} else {
		return false
	}
}
