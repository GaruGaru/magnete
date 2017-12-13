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

	var httpClient = &http.Client{
		Timeout: 16 * time.Second,
	}

	var searchUrl = fmt.Sprintf("%s/search?f=%s", t.url, url.QueryEscape(query))

	return t.torrentList(*httpClient, searchUrl, torrentListMatcher)
}

func (t Torrentz) getRoot(httpClient http.Client, url string) (*html.Node, error) {

	req, err := http.NewRequest("GET", url, nil)
	req.Close = true
	if err != nil {
		log.Fatalln(err)
	}

	req.Close = true

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2228.0 Safari/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	root, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	return root, nil
}

func (t Torrentz) torrentList(httpClient http.Client, url string, matcher scrape.Matcher) []TorrentResult {
	var root, err = t.getRoot(httpClient, url)
	var results []TorrentResult
	if err == nil {
		torrents := scrape.FindAll(root, matcher)
		resultsChannel := make(chan TorrentResult, 1000)
		var providerWaitGroup sync.WaitGroup
		for _, torrentItem := range torrents {
			var title = scrape.Text(torrentItem)
			var itemUrl = scrape.Attr(torrentItem, "href")
			var info = scrape.FindAll(torrentItem.Parent.Parent, sizeMatcher)
			if len(info) == 5 {
				var partial = PartialResult(title, itemUrl, info[2].FirstChild.Data, info[1].FirstChild.Data, info[3].FirstChild.Data, info[4].FirstChild.Data)
				providerWaitGroup.Add(1)
				go t.scrapeItem(httpClient, partial, resultsChannel, &providerWaitGroup)
			} else {
				fmt.Printf("No info for %s\n", itemUrl)
			}
		}

		providerWaitGroup.Wait()
		close(resultsChannel)

		for result := range resultsChannel {
			results = append(results, result)
		}

	} else {
		panic(fmt.Sprintf("Unable to fetch results list for %s: %s", url, err))
	}

	return results
}

func isBlacklisted(provider string) bool {
	return strings.Contains(provider, "btdb.to") // TODO Implement blacklist
}

func (t Torrentz) scrapeItem(httpClient http.Client, item TorrentResult, results chan TorrentResult, providerWaitGroup *sync.WaitGroup) {

	defer providerWaitGroup.Done()

	var magnets, err = t.magnetList(httpClient, fmt.Sprintf("%s%s", t.url, item.Source))

	var magnetWg sync.WaitGroup
	magnetChannel := make(chan TorrentResult, len(magnets))

	if err == nil {
		for _, provider := range magnets {
			if !isBlacklisted(provider) {
				fmt.Printf("Got provider %s for %s\n", provider, item.Title)
				magnetWg.Add(1)
				go scrapeMagnet(magnetWg, t, httpClient, provider, magnetChannel, item)
			} else {
				fmt.Printf("Provider %s for %s: is blacklisted\n", provider, item.Title)
			}
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
func scrapeMagnet(magnetWg sync.WaitGroup, t Torrentz, httpClient http.Client, provider string, magnetChannel chan TorrentResult, item TorrentResult) {
	func() {
		defer magnetWg.Done()
		var magnetUrl, err = t.getMagnent(httpClient, provider)
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
			fmt.Printf("Error on magnet provider %s for %s: %s\n", provider, item.Title, err)
		}
	}()
}

func (t Torrentz) getMagnent(httpClient http.Client, url string) (string, error) {
	var root, err = t.getRoot(httpClient, url)
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

func (t Torrentz) magnetList(httpClient http.Client, url string) ([]string, error) {
	var root, err = t.getRoot(httpClient, url)
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
