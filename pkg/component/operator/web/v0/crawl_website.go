package web

import (
	"context"
	"fmt"
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

// PageInfo defines the information of a page
type PageInfo struct {
	// Link: The URL of the page.
	Link string `json:"link"`
	// Title: The title of the page.
	Title string `json:"title"`
}

// CrawlWebsiteInput defines the input of the scrape website task
type CrawlWebsiteInput struct {
	// URL: The URL of the website to scrape.
	URL string `json:"url"`
	// AllowedDomains: The list of allowed domains to scrape.
	AllowedDomains []string `json:"allowed-domains"`
	// MaxK: The maximum number of pages to scrape.
	MaxK int `json:"max-k"`
	// Timeout: The number of milliseconds to wait before scraping the web page. Min 0, Max 60000.
	Timeout int `json:"timeout"`
	// MaxDepth: The maximum depth of the pages to scrape.
	MaxDepth int `json:"max-depth"`
	// Filter: The filter to filter the URLs to crawl.
	Filter Filter `json:"filter"`
}

// Filter defines the filter of the crawl website task
type Filter struct {
	// ExcludePatterns: The patterns to exclude the URLs to crawl.
	ExcludePatterns []string `json:"exclude-patterns"`
	// IncludePatterns: The patterns to include the URLs to crawl.
	IncludePatterns []string `json:"include-patterns"`
	// ExcludeSubstrings: The substrings to exclude the URLs to crawl.
	ExcludeSubstrings []string `json:"exclude-substrings"`
}

func (i *CrawlWebsiteInput) preset() {
	if i.MaxK <= 0 {
		// When the users set to 0, it means infinite.
		// However, there is performance issue when we set it to infinite.
		// The issue may come from the conflict of goruntine and colly library.
		// We have not targeted the specific reason of the issue.
		// We set 120 seconds as the timeout in CrawlSite function.
		// After testing, we found that we can crawl around 8000 pages in 120 seconds.
		// So, we set the default value to solve performance issue easily.
		i.MaxK = 8000
	}
}

// ScrapeWebsiteOutput defines the output of the scrape website task
type ScrapeWebsiteOutput struct {
	// Pages: The list of pages that were scraped.
	Pages []PageInfo `json:"pages"`
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// CrawlWebsite navigates through a website and return the links and titles of the pages
func (e *execution) CrawlWebsite(input *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := CrawlWebsiteInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("error converting input to struct: %v", err)
	}

	inputStruct.preset()

	output := ScrapeWebsiteOutput{
		Pages: []PageInfo{},
	}

	if !targetLink(inputStruct.URL, inputStruct.Filter) {
		outputStruct, err := base.ConvertToStructpb(output)
		if err != nil {
			return nil, fmt.Errorf("convert output to structpb error: %v", err)
		}
		return outputStruct, nil
	}

	c := initColly(inputStruct)

	var mu sync.Mutex
	pageLinks := []string{}

	// We will have the component timeout feature in the future.
	// Before that, we initialize the context here.
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// On every a element which has href attribute call callback
	// Wont be called if error occurs
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()

		if ctx.Err() != nil {
			return
		}

		// If we set output.Pages to the slice of PageInfo, it will take a longer time if the first html page has a lot of links.
		// To improve the small Max-K execution time, we will use a separate slice to store the links.
		// However, when K is big, the output length could be less than K.
		// So, I set twice the MaxK to stop the scraping.
		if len(pageLinks) >= inputStruct.MaxK*getPageTimes(inputStruct.MaxK) {
			return
		}

		link := e.Attr("href")
		absoluteURL := e.Request.AbsoluteURL(link)
		if !targetLink(absoluteURL, inputStruct.Filter) {
			return
		}

		if util.InSlice(pageLinks, link) {
			return
		}

		pageLinks = append(pageLinks, link)

		_ = e.Request.Visit(link)
	})

	// Set error handler
	c.OnError(func(r *colly.Response, err error) {
		// In the future, we can design the error handling logic.
	})

	c.OnRequest(func(r *colly.Request) {
		mu.Lock()
		defer mu.Unlock()

		// Before length of output page is over, we should always send request.
		if (len(output.Pages) >= inputStruct.MaxK) || ctx.Err() != nil {
			r.Abort()
			return
		}
		// Set a random user agent to avoid being blocked by websites
		r.Headers.Set("User-Agent", randomString())
	})

	// colly.Wait() does not terminate the program. So, we need a system to terminate the program when there is no collector.
	// We use a channel to notify the main goroutine that a new page has been scraped.
	// When there is no new page for 2 seconds, we cancel the context.
	pageUpdateCh := make(chan struct{})

	c.OnResponse(func(r *colly.Response) {
		if ctx.Err() != nil {
			return
		}

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

		title := util.ScrapeWebpageTitle(doc)
		page.Title = title

		mu.Lock()
		defer mu.Unlock()
		// If we do not set this condition, the length of output.Pages could be over the limit.
		if len(output.Pages) < inputStruct.MaxK {
			output.Pages = append(output.Pages, page)

			// Signal that we've added a new page
			select {
			case pageUpdateCh <- struct{}{}:
			default:
			}

			// If the length of output.Pages is equal to MaxK, we should stop the scraping.
			if len(output.Pages) == inputStruct.MaxK {
				cancel()
				return
			}
			return
		}
		cancel()

	})

	// Start scraping
	if !strings.HasPrefix(inputStruct.URL, "http://") && !strings.HasPrefix(inputStruct.URL, "https://") {
		inputStruct.URL = "https://" + inputStruct.URL
	}

	go func() {
		_ = c.Visit(inputStruct.URL)
		c.Wait()
	}()

	// To avoid to wait for 2 minutes, we use a timer to check if there is a new page.
	// If there is no new page, we cancel the context.
	go func() {
		inactivityTimer := time.NewTimer(2 * time.Second)
		defer inactivityTimer.Stop()

		for {
			select {
			case <-pageUpdateCh:
				// Reset the timer whenever we get a new page
				if !inactivityTimer.Stop() {
					select {
					case <-inactivityTimer.C:
					default:
					}
				}
				inactivityTimer.Reset(2 * time.Second)
			case <-inactivityTimer.C:
				// If no new pages for 2 seconds, cancel the context
				cancel()
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	<-ctx.Done()

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
	var parallel int
	if inputStruct.MaxK < 30 {
		parallel = inputStruct.MaxK
	} else {
		parallel = 30
	}

	_ = c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: parallel,
		// We set the delay to avoid being blocked.
		Delay: 100 * time.Millisecond,
	})

	// Timeout here is set of each page rather than whole colly instance.
	c.SetRequestTimeout(time.Duration(inputStruct.Timeout) * time.Millisecond)

	if len(inputStruct.AllowedDomains) > 0 {
		c.AllowedDomains = inputStruct.AllowedDomains
	}
	c.AllowURLRevisit = false

	return c
}

// It ensures that we fetch enough pages to get the required number of pages.
func getPageTimes(maxK int) int {
	if maxK < 10 {
		return 30
	} else {
		return 3
	}
}
