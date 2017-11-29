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
	"log"
)

type Torrentz struct {
	url     string
	timeout time.Duration
}

func NewTorrentz(url string, timeout time.Duration) Torrentz {
	return Torrentz{url, timeout}
}

func (t Torrentz) Get(query string) []TorrentResult {
	var searchUrl = fmt.Sprintf("%s/search?f=%s", t.url, url.QueryEscape(query))
	return t.torrentList(searchUrl, torrentListMatcher)
}

func (t Torrentz) getRoot(url string) (*html.Node, error) {

	var transport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: time.Second,
		}).Dial,
		TLSHandshakeTimeout: time.Second,
	}

	var httpClient = &http.Client{
		Timeout:   t.timeout,
		Transport: transport,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalln(err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2228.0 Safari/537.36")

	resp, err := httpClient.Do(req)
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
	var root, err = t.getRoot(url)
	var results []TorrentResult
	if err == nil {

		torrents := scrape.FindAll(root, matcher)

		resultsChannel := make(chan TorrentResult, 1000)
		var wg sync.WaitGroup
		for _, torrentItem := range torrents {
			var title = scrape.Text(torrentItem)
			var itemUrl = scrape.Attr(torrentItem, "href")
			var info = scrape.FindAll(torrentItem.Parent.Parent, sizeMatcher)
			if len(info) == 5 {
				wg.Add(1)
				var partial = PartialResult(title, itemUrl, info[2].FirstChild.Data, info[1].FirstChild.Data, info[3].FirstChild.Data, info[4].FirstChild.Data)
				go t.scrapeItem(partial, resultsChannel, &wg)
			} else {
				fmt.Printf("No info for %s: %s\n", itemUrl, err)
			}
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

func (t Torrentz) scrapeItem(item TorrentResult, results chan TorrentResult, wg *sync.WaitGroup) {
	defer wg.Done()
	var magnets, err = t.magnetList(fmt.Sprintf("%s%s", t.url, item.Source))

	var magnetWg sync.WaitGroup
	magnetChannel := make(chan TorrentResult, len(magnets))

	if err == nil {
		for _, m := range magnets {
			magnetWg.Add(1)
			go func() {
				defer magnetWg.Done()
				var magnetUrl, err = t.getMagnent(m)
				if err == nil {
					magnetChannel <- TorrentResult{
						Title:  item.Title,
						Source: item.Source,
						Magnet: magnetUrl,
						Size:   item.Size,
						Peers:  item.Peers,
						Seeds:  item.Seeds,
						Age:    item.Age,
					}
				} else {
					fmt.Printf("Error on magnet provider %s for %s: %s\n", m, item.Title, err)
				}
			}()
		}

		magnetWg.Wait()
		close(magnetChannel)

		if len(magnetChannel) != 0 {
			var last TorrentResult
			for result := range magnetChannel {
				last = result
			}
			fmt.Printf("[OK] Got magnet for %s\n", item.Title)
			results <- last
		} else {
			fmt.Printf("No magnet found for %s\n", item.Title)
		}

	} else {
		fmt.Printf("Can't get magnet provider list for %s: %s\n", item.Source, err)
	}
}

func (t Torrentz) getMagnent(url string) (string, error) {
	var root, err = t.getRoot(url)
	if err != nil {
		return "", err
	}
	var urls = scrape.FindAll(root, magnetUrlMatcher)
	if len(urls) > 0 {
		return scrape.Attr(urls[0], "href"), nil
	} else {
		return "no_magnet", fmt.Errorf("no Magnet found in %s: %s", url, err)
	}
}

func (t Torrentz) magnetList(url string) ([]string, error) {
	var root, err = t.getRoot(url)
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

func sizeMatcher(n *html.Node) bool {
	return n.DataAtom == atom.Span
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
