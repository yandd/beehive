// Package crawlerbee is a Bee for crawlering web to feeds.
package crawlerbee

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/muesli/beehive/bees"
)

// CrawlerBee is a Bee for handling Web feeds.
type CrawlerBee struct {
	bees.Bee

	url string
	// decides whether the next fetch should be skipped
	skipNextFetch bool

	feedSel        string
	titleSel       string
	descriptionSel string
	urlSel         string

	fullCompare bool

	lastTitle  string
	lastTitleM map[string]struct{}

	eventChan chan bees.Event
}

func (mod *CrawlerBee) pollFeed(uri string, timeout int) {
	wait := time.Duration(0)
	for {
		select {
		case <-mod.SigChan:
			return

		case <-time.After(wait):
			if err := mod.Fetch(timeout); err != nil {
				mod.LogErrorf("%s: %s", uri, err)
			}
		}

		wait = 5 * time.Minute
	}
}

// Run executes the Bee's event loop.
func (mod *CrawlerBee) Run(cin chan bees.Event) {
	mod.eventChan = cin

	time.Sleep(10 * time.Second)
	mod.pollFeed(mod.url, 10)
}

// ReloadOptions parses the config options and initializes the Bee.
func (mod *CrawlerBee) ReloadOptions(options bees.BeeOptions) {
	mod.SetOptions(options)

	options.Bind("skip_first", &mod.skipNextFetch)
	options.Bind("url", &mod.url)
	options.Bind("feed_sel", &mod.feedSel)
	options.Bind("url_sel", &mod.urlSel)
	options.Bind("title_sel", &mod.titleSel)
	options.Bind("description_sel", &mod.descriptionSel)
	options.Bind("full_compare", &mod.fullCompare)
}

func (mod *CrawlerBee) Fetch(timeout int) error {
	site, err := url.Parse(mod.url)
	if err != nil {
		return fmt.Errorf("url: \"%s\"invalid", mod.url)
	}

	if !site.IsAbs() {
		return fmt.Errorf("url: \"%s\" invalid", mod.url)
	}

	transport := http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, time.Duration(timeout)*time.Second)
		},
	}

	client := http.Client{
		Transport: &transport,
	}

	resp, err := client.Get(mod.url)
	if err != nil {
		return err
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return err
	}

	feedSel := doc.Find(mod.feedSel)
	if feedSel.Length() < 1 {
		return fmt.Errorf("feed_sel: length %d", feedSel.Length())
	}

	var errInEach error

	lastTitle := mod.lastTitle
	lastTitleM := map[string]struct{}{}

	feedCnt := 0

	feedSel.EachWithBreak(func(idx int, sel *goquery.Selection) bool {
		titleSel := sel.Find(mod.titleSel)
		if titleSel == nil {
			errInEach = fmt.Errorf("title_sel: invalid")
			return false
		}

		if titleSel.Length() < 1 {
			errInEach = fmt.Errorf("title_sel: length %d", titleSel.Length())
			return false
		}

		title := titleSel.First().Text()

		if len(title) == 0 {
			errInEach = fmt.Errorf("title_sel: invalid, title's lenght eq 0")
			return false
		}

		lastTitleM[title] = struct{}{}

		if !mod.fullCompare {
			if title == lastTitle {
				return false
			}
		} else {
			if _, ok := mod.lastTitleM[title]; ok {
				return true
			}
		}

		uri := ""
		if len(mod.urlSel) != 0 {
			urlSel := sel.Find(mod.urlSel)
			if urlSel == nil {
				errInEach = fmt.Errorf("url_sel: invalid")
				return false
			}

			if urlSel.Length() < 1 {
				errInEach = fmt.Errorf("url_sel: length %d", urlSel.Length())
				return false
			}

			urlStr, ok := urlSel.First().Attr("href")
			if !ok {
				errInEach = fmt.Errorf("url_sel: invalid")
				return false
			}

			if len(urlStr) == 0 {
				errInEach = fmt.Errorf("url_sel: invalid")
				return false
			}

			u, err := url.Parse(urlStr)
			if err != nil {
				errInEach = fmt.Errorf("url_sel: invalid")
				return false
			}

			if !u.IsAbs() {
				u = site.ResolveReference(u)
				urlStr = u.String()
			}

			uri = urlStr
		}

		description := ""
		if len(mod.descriptionSel) != 0 {
			descriptionSel := sel.Find(mod.descriptionSel)
			if descriptionSel == nil {
				errInEach = fmt.Errorf("description_sel: invalid")
				return false
			}

			if descriptionSel.Length() < 1 {
				errInEach = fmt.Errorf("description_sel: length %d", descriptionSel.Length())
				return false
			}

			description = descriptionSel.First().Text()
		}

		if idx == 0 {
			mod.lastTitle = title
		}

		if mod.skipNextFetch {
			mod.skipNextFetch = false
			return false
		}

		newitemEvent := bees.Event{
			Bee:  mod.Name(),
			Name: "new_item",
			Options: []bees.Placeholder{
				{
					Name:  "title",
					Type:  "string",
					Value: title,
				},
				{
					Name:  "description",
					Type:  "string",
					Value: description,
				},
				{
					Name:  "url",
					Type:  "string",
					Value: uri,
				},
			},
		}

		mod.eventChan <- newitemEvent

		feedCnt++

		return true
	})

	if errInEach != nil {
		return errInEach
	}

	if mod.fullCompare {
		mod.lastTitleM = lastTitleM
	}

	if feedCnt > 0 {
		if !mod.fullCompare {
			mod.Logf("%d new item(s) in %s, last title \"%s\"", feedCnt, mod.url, mod.lastTitle)
		} else {
			mod.Logf("%d new item(s) in %s, last title map \"%v\"", feedCnt, mod.url, mod.lastTitleM)
		}
	}

	return nil
}
