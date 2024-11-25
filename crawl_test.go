package ssg

import (
	"strings"
	"testing"
)

func TestCrawler(t *testing.T) {
	urlsAndMatchedTexts := map[string]string{
		"https://github.com/meinside":                       "It's me inside me.",
		"https://x.com/elonmusk/status/1813223978884079966": "Build a Community around anything at all on 𝕏!",
	}

	urls := []string{}
	for k := range urlsAndMatchedTexts {
		urls = append(urls, k)
	}

	if client, err := NewScrapper(); err == nil {
		client.SetSelectorReturner(_selectorReturner)

		if crawled, err := client.CrawlURLs(urls, false); err == nil {
			if len(crawled) != len(urls) {
				t.Errorf("the count of crawled urls: %d does not match with the input: %d", len(crawled), len(urls))
			}

			for url, txt := range crawled {
				if !strings.Contains(txt, urlsAndMatchedTexts[url]) {
					t.Errorf("crawled content from '%s' does not contain expected text '%s'", url, urlsAndMatchedTexts[url])
				}
			}
		} else {
			t.Errorf("failed to crawl urls: %s", err)
		}

		if err := client.Close(); err != nil {
			t.Errorf("failed to close scrapper: %s", err)
		}
	} else {
		t.Errorf("failed to create a new scrapper: %s", err)
	}
}

func TestCrawlerWithURLReplacer(t *testing.T) {
	url := "https://www.reddit.com/r/IAmA/comments/2rgsan/i_am_elon_musk_ceocto_of_a_rocket_company_ama/"
	matched := "I am Elon Musk, CEO/CTO of a rocket company, AMA! "

	if client, err := NewScrapper(); err == nil {
		client.SetURLReplacer(_urlReplacer)

		if crawled, err := client.CrawlURLs([]string{url}, false); err == nil {
			if len(crawled) != 1 {
				t.Errorf("the count of crawled urls: %d does not match with the input: %d", len(crawled), 1)
			}

			replacedURL := _urlReplacer(url)
			if txt, exists := crawled[replacedURL]; exists {
				if !strings.Contains(txt, matched) {
					t.Errorf("crawled content from '%s' does not contain expected text '%s'", url, matched)
				}
			} else {
				t.Errorf("no replaced url '%s' exists in the crawled result", replacedURL)
			}
		} else {
			t.Errorf("failed to crawl urls: %s", err)
		}

		if err := client.Close(); err != nil {
			t.Errorf("failed to close scrapper: %s", err)
		}
	} else {
		t.Errorf("failed to create a new scrapper: %s", err)
	}
}

// NOTE: return `div[data-testid='tweetText'] > span` for `x.com`
func _selectorReturner(url string) string {
	if strings.HasPrefix(url, "https://x.com") {
		return `article[data-testid='tweet']`
	}
	return `body`
}

// NOTE: use `old.reddit.com` instead of `www.reddit.com`
func _urlReplacer(url string) string {
	if strings.HasPrefix(url, "https://www.reddit.com/") {
		return strings.Replace(url, "www.reddit.com", "old.reddit.com", 1)
	}
	return url
}
