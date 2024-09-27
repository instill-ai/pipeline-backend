package googlesearch

import (
	"fmt"
	"log"
	"net/http"

	"github.com/PuerkitoBio/goquery"
	"google.golang.org/api/customsearch/v1"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

const (
	// MaxResultsPerPage is the default max number of search results per page
	MaxResultsPerPage = 10
	// MaxResults is the maximum number of search results
	MaxResults = 100
)

// Min returns the smaller of x or y.
func Min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

// SearchInput defines the input of the search task
type SearchInput struct {
	// Query: The search query.
	Query string `json:"query"`

	// TopK: The number of search results to return.
	TopK *int `json:"top-k,omitempty"`

	// IncludeLinkText: Whether to include the scraped text of the search web page result.
	IncludeLinkText *bool `json:"include-link-text,omitempty"`

	// IncludeLinkHTML: Whether to include the scraped HTML of the search web page result.
	IncludeLinkHTML *bool `json:"include-link-html,omitempty"`
}

type Result struct {
	// Title: The title of the search result, in plain text.
	Title string `json:"title"`

	// Link: The full URL to which the search result is pointing, e.g.
	// http://www.example.com/foo/bar.
	Link string `json:"link"`

	// Snippet: The snippet of the search result, in plain text.
	Snippet string `json:"snippet"`

	// LinkText: The scraped text of the search web page result, in plain text.
	LinkText string `json:"link-text"`

	// LinkHTML: The full raw HTML of the search web page result.
	LinkHTML string `json:"link-html"`
}

// SearchOutput defines the output of the search task
type SearchOutput struct {
	// Results: The search results.
	Results []*Result `json:"results"`
}

// Scrape the search results if needed
func scrapeSearchResults(searchResults *customsearch.Search, includeLinkText, includeLinkHTML bool) ([]*Result, error) {
	results := []*Result{}
	for _, item := range searchResults.Items {
		linkText, linkHTML := "", ""
		if includeLinkText || includeLinkHTML {
			// Make an HTTP GET request to the web page
			client := &http.Client{Transport: &http.Transport{
				DisableKeepAlives: true,
			}}
			response, err := client.Get(item.Link)
			if err != nil {
				log.Printf("Error making HTTP GET request to %s: %v", item.Link, err)
				continue
			}
			defer response.Body.Close()

			// Parse the HTML content
			doc, err := goquery.NewDocumentFromReader(response.Body)
			if err != nil {
				fmt.Printf("Error parsing %s: %v", item.Link, err)
			}

			if includeLinkHTML {
				linkHTML, err = util.ScrapeWebpageHTML(doc)
				if err != nil {
					log.Printf("Error scraping HTML from %s: %v", item.Link, err)
				}
			}

			if includeLinkText {
				linkHTML, err = util.ScrapeWebpageHTML(doc)
				if err != nil {
					log.Printf("Error scraping HTML from %s: %v", item.Link, err)
				}

				domain, err := util.GetDomainFromURL(item.Link)
				if err != nil {
					log.Printf("Error getting domain from %s: %v", item.Link, err)
				}

				linkText, err = util.ScrapeWebpageHTMLToMarkdown(linkHTML, domain)
				if err != nil {
					log.Printf("Error scraping text from %s: %v", item.Link, err)
				}
			}

		}

		results = append(results, &Result{
			Title:    item.Title,
			Link:     item.Link,
			Snippet:  item.Snippet,
			LinkText: linkText,
			LinkHTML: linkHTML,
		})
	}
	return results, nil
}

// Search the web using Google Custom Search API and scrape the results if needed
func search(cseListCall *customsearch.CseListCall, input SearchInput) (SearchOutput, error) {
	output := SearchOutput{}

	if input.TopK == nil {
		defaultTopK := int(MaxResultsPerPage)
		input.TopK = &defaultTopK
	}
	if *input.TopK <= 0 || int64(*input.TopK) > MaxResults {
		return output, fmt.Errorf("top-k must be between 1 and %d", MaxResults)
	}

	if input.IncludeLinkHTML == nil {
		defaultValue := false
		input.IncludeLinkHTML = &defaultValue
	}
	if input.IncludeLinkText == nil {
		defaultValue := false
		input.IncludeLinkText = &defaultValue
	}

	// Make the search request
	results := []*Result{}

	for start := 1; start <= *input.TopK; start += MaxResultsPerPage {
		searchNum := Min(*input.TopK-start+1, MaxResultsPerPage)
		searchResults, err := cseListCall.Q(input.Query).Start(int64(start)).Num(int64(searchNum)).Do()
		if err != nil {
			return output, err
		}
		rs, err := scrapeSearchResults(searchResults, *input.IncludeLinkText, *input.IncludeLinkHTML)
		if err != nil {
			return output, err
		}
		results = append(results, rs...)
	}
	output.Results = results

	return output, nil
}
