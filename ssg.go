package ssg

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/playwright-community/playwright-go"

	rsg "github.com/meinside/randomized-string-generator-go"
)

const (
	randomUserAgentPattern = `Mozilla/{{}} (Macintosh; Intel Mac OS X {{}}; rv:{{}}) Gecko/{{}} Firefox/{{}}`
)

// Scrapper struct
type Scrapper struct {
	userAgentGenerator *rsg.Randomizer
	fixedUserAgent     string

	pw      *playwright.Playwright
	browser playwright.Browser

	urlReplacer         func(from string) string
	htmlElementsRemover func(doc *goquery.Document)
	plainTextTidier     func(str string) string
}

// NewScrapper creates a new scrapper client.
func NewScrapper() (s *Scrapper, err error) {
	if err = playwright.Install(); err != nil {
		return nil, fmt.Errorf("failed to install playwright: %s", err)
	}

	var _pw *playwright.Playwright
	if _pw, err = playwright.Run(); err == nil {
		var _browser playwright.Browser
		if _browser, err = _pw.Firefox.Launch(); err == nil {
			return &Scrapper{
				userAgentGenerator: rsg.MustCompile(
					randomUserAgentPattern,
					rsg.RandomVersionMajorMinor(
						0, 0, // major (ignored: using fixed value below = 5)
						0, 10, // minor = [0, 10)

						// fixed
						5, // major = 5
					),
					rsg.RandomVersionMajorMinor(
						0, 0, // major (ignored: using fixed value below = 10)
						15, 20, // minor = [15, 20)

						// fixed
						10, // major = 10
					),
					rsg.RandomVersionMajorMinor(
						100, 200, // major = [100, 200)
						0, 0, // minor (ignored: using fixed value below = 0)

						// fixed
						-1, // < 0, ignored
						0,  // minor = 0
					),
					rsg.RandomYYYYMMDD(
						2010,   // from 2010,
						365*14, // within 14 years
					),
					rsg.RandomVersionMajorMinor(
						100, 200, // major = [100, 200)
						0, 0, // minor (ignored: using fixed value below = 0)

						// fixed
						-1, // < 0, ignored
						0,  // minor = 0
					),
				),

				pw:      _pw,
				browser: _browser,

				urlReplacer:         defaultURLReplacer,
				htmlElementsRemover: defaultHTMLElementsRemover,
				plainTextTidier:     defaultPlainTextTidier,
			}, nil
		} else {
			_ = _pw.Stop()
		}
	}

	return nil, err
}

// SetFixedUserAgent sets the fixed user-agent string for the scrapper client.
func (s *Scrapper) SetFixedUserAgent(userAgent string) {
	s.fixedUserAgent = userAgent
}

// SetURLReplacer sets the url replacer function for the scrapper client.
func (s *Scrapper) SetURLReplacer(replacer func(from string) (to string)) {
	s.urlReplacer = replacer
}

// SetHTMLElementRemover sets the HTML element remover function for the scrapper client.
func (s *Scrapper) SetHTMLElementRemover(remover func(doc *goquery.Document)) {
	s.htmlElementsRemover = remover
}

// SetPlainTextTidier sets the plain text tidier function for the scrapper client.
func (s *Scrapper) SetPlainTextTidier(tidier func(str string) string) {
	s.plainTextTidier = tidier
}

// CrawlURLs crawls contents from given `urls`.
func (s *Scrapper) CrawlURLs(urls []string, asHTML bool) (crawled map[string]string, err error) {
	crawled = map[string]string{}
	errs := []error{}

	var userAgent string
	if len(s.fixedUserAgent) > 0 {
		userAgent = s.fixedUserAgent
	} else {
		userAgent = s.userAgentGenerator.Generate() // randomized user-agent
	}

	var ctx playwright.BrowserContext
	if ctx, err = s.browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(userAgent),
	}); err == nil {
		var page playwright.Page
		if page, err = ctx.NewPage(); err == nil {
			var parsedURL *url.URL
			var referrer, html string
			for _, u := range urls {
				// replace given url if `urlReplacer` is set for the client
				if s.urlReplacer != nil {
					u = s.urlReplacer(u)
				}

				if parsedURL, err = url.Parse(u); err == nil {
					referrer = parsedURL.Scheme + "://" + parsedURL.Host
					if _, err = page.Goto(u, playwright.PageGotoOptions{
						Referer: playwright.String(referrer),
					}); err == nil {
						if html, err = page.Content(); err == nil {
							doc, _ := goquery.NewDocumentFromReader(bytes.NewBuffer([]byte(html)))

							// remove unwanted HTML elements
							if s.htmlElementsRemover != nil {
								s.htmlElementsRemover(doc)
							}

							if asHTML { // return as HTML
								if html, err = doc.Html(); err == nil {
									crawled[u] = html
								} else {
									errs = append(errs, fmt.Errorf("failed to get content of page '%s' as HTML: %s", u, err))
								}
							} else { // return as plain-text
								crawled[u] = doc.Text()

								// tidy plain text
								if s.plainTextTidier != nil {
									crawled[u] = s.plainTextTidier(crawled[u])
								}
							}
						} else {
							errs = append(errs, fmt.Errorf("failed to get content of page '%s': %s", u, err))
						}
					} else {
						errs = append(errs, fmt.Errorf("failed to go to page '%s': %s", u, err))
					}
				} else {
					errs = append(errs, fmt.Errorf("failed to parse url '%s': %s", u, err))
				}
			}
		} else {
			errs = append(errs, fmt.Errorf("failed to create page: %s", err))
		}
	} else {
		errs = append(errs, fmt.Errorf("failed to create browser context: %s", err))
	}

	if len(errs) > 0 {
		err = errors.Join(errs...)
	}

	return crawled, err
}

// Close closes the scrapper client.
func (s *Scrapper) Close() (err error) {
	errs := []error{}

	err = s.browser.Close()
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to close browser: %s", err))
	}

	err = s.pw.Stop()
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to stop playwright: %s", err))
	}

	if len(errs) > 0 {
		err = errors.Join(errs...)
	}

	return err
}

// CrawlURLs crawls contents from given `urls`.
//
// Pass `userAgent` as an empty string for generating randomized user-agent string.
//
// It is just a helper function for convenience.
func CrawlURLs(userAgent string, urls []string, asHTML bool) (crawled map[string]string, err error) {
	crawled = map[string]string{}
	errs := []error{}

	var client *Scrapper
	if client, err = NewScrapper(); err == nil {
		if len(userAgent) > 0 {
			client.SetFixedUserAgent(userAgent)
		}

		// close things
		defer func() {
			if err = client.Close(); err != nil {
				log.Printf("failed to close client: %s", err)
			}
		}()

		return client.CrawlURLs(urls, asHTML)
	} else {
		errs = append(errs, fmt.Errorf("failed to create scrapper client: %s", err))
	}

	if len(errs) > 0 {
		err = errors.Join(errs...)
	}

	return crawled, err
}

// replace given url `from`, actually doing nothing (default)
func defaultURLReplacer(from string) string {
	return from
}

// remove unwanted HTML elements from given document (default)
func defaultHTMLElementsRemover(doc *goquery.Document) {
	_ = doc.Find("head").Remove()                     // head
	_ = doc.Find("script").Remove()                   // javascripts
	_ = doc.Find("noscript").Remove()                 // noscripts
	_ = doc.Find("link[rel=\"stylesheet\"]").Remove() // css links
	_ = doc.Find("style").Remove()                    // embeded css styles
	_ = doc.Find("meta").Remove()                     // metas
}

// compact given plain text (default)
func defaultPlainTextTidier(str string) string {
	// trim each line
	trimmed := []string{}
	for _, line := range strings.Split(str, "\n") {
		trimmed = append(trimmed, strings.TrimRight(line, " "))
	}
	str = strings.Join(trimmed, "\n")

	// remove redundant empty lines
	regex := regexp.MustCompile("\n{2,}")
	return regex.ReplaceAllString(str, "\n")
}