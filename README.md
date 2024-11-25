# simple-scrapper-go

A simple web scrapper written in Golang.

Fetch contents from URLs with [headless browsers](https://github.com/playwright-community/playwright-go), and remove unwanted elements from them with [goquery](https://github.com/PuerkitoBio/goquery) & etc..

## Usage

```go
package main

import (
	"log"
	"strings"

	ssg "github.com/meinside/simple-scrapper-go"
)

const (
	returnAsHTML = true
)

func main() {
	urls := []string{
		"https://x.com/elonmusk/status/1813223978884079966",
		"https://www.reddit.com/r/IAmA/comments/2rgsan/i_am_elon_musk_ceocto_of_a_rocket_company_ama/",
	}

	// create client
	if client, err := ssg.NewScrapper(); err == nil {
		client.SetURLReplacer(func(url string) string {
			// NOTE: use `old.reddit.com` instead of `www.reddit.com`
			if strings.HasPrefix(url, "https://www.reddit.com/") {
				return strings.Replace(url, "www.reddit.com", "old.reddit.com", 1)
			}
			return url
		})
		client.SetSelectorReturner(func(url string) string {
			// NOTE: select `div[data-testid="tweetText"]` for `x.com`
			if strings.Contains(url, "x.com/") {
				return `div[data-testid="tweetText"]`
			}
			return `body`
		})

		// crawl urls
		if crawled, err := client.CrawlURLs(urls, returnAsHTML); err == nil {
			log.Printf("crawled things: %+v", crawled)
		} else {
			log.Printf("failed to crawl urls: %s", err)
		}

		// close client
		if err := client.Close(); err != nil {
			log.Printf("failed to close scrapper: %s", err)
		}
	} else {
		log.Printf("failed to create scrapper: %s", err)
	}
}
```

## Known Issues

- [ ] Scrapping reddit's URLs often fails.

## License

MIT

