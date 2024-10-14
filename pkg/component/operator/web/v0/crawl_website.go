package web

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/rand"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/PuerkitoBio/goquery"

	colly "github.com/gocolly/colly/v2"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

type PageInfo struct {
	Link     string `json:"link"`
	Title    string `json:"title"`
	LinkText string `json:"link-text"`
	LinkHTML string `json:"link-html"`
}

// CrawlWebsiteInput defines the input of the scrape website task
type CrawlWebsiteInput struct {
	// TargetURL: The URL of the website to scrape.
	TargetURL string `json:"target-url"`
	// AllowedDomains: The list of allowed domains to scrape.
	AllowedDomains []string `json:"allowed-domains"`
	// MaxK: The maximum number of pages to scrape.
	MaxK int `json:"max-k"`
	// IncludeLinkText: Whether to include the scraped text of the scraped web page.
	IncludeLinkText *bool `json:"include-link-text"`
	// IncludeLinkHTML: Whether to include the scraped HTML of the scraped web page.
	IncludeLinkHTML *bool `json:"include-link-html"`
	// OnlyMainContent: Whether to scrape only the main content of the web page. If true, the scraped text wull exclude the header, nav, footer.
	OnlyMainContent bool `json:"only-main-content"`
	// RemoveTags: The list of tags to remove from the scraped text.
	RemoveTags []string `json:"remove-tags"`
	// OnlyIncludeTags: The list of tags to include in the scraped text.
	OnlyIncludeTags []string `json:"only-include-tags"`
	// Timeout: The number of milliseconds to wait before scraping the web page. Min 0, Max 60000.
	Timeout int `json:"timeout"`
	// MaxDepth: The maximum depth of the pages to scrape.
	MaxDepth int `json:"max-depth"`
}

func (i *CrawlWebsiteInput) Preset() {
	if i.IncludeLinkHTML == nil {
		b := false
		i.IncludeLinkHTML = &b
	}
	if i.IncludeLinkText == nil {
		b := false
		i.IncludeLinkText = &b
	}
	if i.MaxK < 0 {
		i.MaxK = 0
	}
}

// ScrapeWebsiteOutput defines the output of the scrape website task
type ScrapeWebsiteOutput struct {
	// Pages: The list of pages that were scraped.
	Pages []PageInfo `json:"pages"`
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// Scrape crawls a webpage and returns a slice of PageInfo
func (e *execution) CrawlWebsite(input *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := CrawlWebsiteInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("error converting input to struct: %v", err)
	}

	inputStruct.Preset()

	output := ScrapeWebsiteOutput{}

	c := initColly(inputStruct)

	var mu sync.Mutex
	pageLinks := []string{}

	// On every a element which has href attribute call callback
	// Wont be called if error occurs
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		// If we set output.Pages to the slice of PageInfo, it will take a longer time if the first html page has a lot of links.
		// To improve the small Max-K execution time, we will use a separate slice to store the links.
		// However, when K is big, the output length could be less than K.
		// So, I set twice the MaxK to stop the scraping.
		if inputStruct.MaxK > 0 && len(pageLinks) >= inputStruct.MaxK*2 {
			return
		}

		link := e.Attr("href")

		if util.InSlice(pageLinks, link) {
			return
		}

		pageLinks = append(pageLinks, link)

		_ = e.Request.Visit(link)
	})

	// Set error handler
	c.OnError(func(r *colly.Response, err error) {
		log.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	c.OnRequest(func(r *colly.Request) {
		// Before length of output page is over, we should always send request.
		if inputStruct.MaxK > 0 && len(output.Pages) >= inputStruct.MaxK {
			r.Abort()
			return
		}

		// Set a random user agent to avoid being blocked by websites
		r.Headers.Set("User-Agent", randomString())
	})

	c.OnResponse(func(r *colly.Response) {

		strippedURL := stripQueryAndTrailingSlash(r.Request.URL)

		page := PageInfo{}

		page.Link = strippedURL.String()

		html := string(r.Body)
		ioReader := strings.NewReader(html)
		doc, err := goquery.NewDocumentFromReader(ioReader)

		if err != nil {
			fmt.Printf("Error parsing %s: %v", strippedURL.String(), err)
			return
		}

		html = getRemovedTagsHTML(doc, inputStruct)

		title := util.ScrapeWebpageTitle(doc)
		page.Title = title

		if *inputStruct.IncludeLinkHTML {
			page.LinkHTML = html
		}

		if *inputStruct.IncludeLinkText {
			domain, err := util.GetDomainFromURL(strippedURL.String())

			if err != nil {
				log.Printf("Error getting domain from %s: %v", strippedURL.String(), err)
				return
			}

			markdown, err := util.ScrapeWebpageHTMLToMarkdown(html, domain)

			if err != nil {
				log.Printf("Error scraping text from %s: %v", strippedURL.String(), err)
				return
			}

			page.LinkText = markdown
		}

		defer mu.Unlock()
		mu.Lock()
		// If we do not set this condition, the length of output.Pages could be over the limit.
		if len(output.Pages) < inputStruct.MaxK {
			output.Pages = append(output.Pages, page)
		}
	})

	// Start scraping
	if !strings.HasPrefix(inputStruct.TargetURL, "http://") && !strings.HasPrefix(inputStruct.TargetURL, "https://") {
		inputStruct.TargetURL = "https://" + inputStruct.TargetURL
	}
	_ = c.Visit(inputStruct.TargetURL)
	c.Wait()

	outputStruct, err := base.ConvertToStructpb(output)
	if err != nil {
		return nil, fmt.Errorf("error converting output to struct: %v", err)
	}

	return outputStruct, nil

}

// randomString generates a random string of length 10-20
func randomString() string {
	b := make([]byte, rand.Intn(10)+10)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// stripQueryAndTrailingSlash removes query parameters and trailing '/' from a URL
func stripQueryAndTrailingSlash(u *url.URL) *url.URL {
	// Remove query parameters by setting RawQuery to an empty string
	u.RawQuery = ""

	// Remove trailing '/' from the path
	u.Path = strings.TrimSuffix(u.Path, "/")

	return u
}

func initColly(inputStruct CrawlWebsiteInput) *colly.Collector {
	c := colly.NewCollector(
		colly.MaxDepth(inputStruct.MaxDepth),
		colly.Async(true),
	)

	// Limit the number of requests to avoid being blocked.
	// Set it to 10 first in case sending too many requests at once.
	var parallel int
	if inputStruct.MaxK < 10 {
		parallel = inputStruct.MaxK
	} else {
		parallel = 10
	}

	_ = c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: parallel,
	})

	c.SetRequestTimeout(time.Duration(inputStruct.Timeout) * time.Millisecond)

	if len(inputStruct.AllowedDomains) > 0 {
		c.AllowedDomains = inputStruct.AllowedDomains
	}
	c.AllowURLRevisit = false

	return c
}
