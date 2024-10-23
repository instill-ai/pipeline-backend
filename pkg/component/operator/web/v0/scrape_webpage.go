package web

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/k3a/html2text"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

// ScrapeWebpagesInput defines the input of the scrape webpage task
type ScrapeWebpagesInput struct {
	// URLs: The URLs of the webpage to scrape.
	URLs []string `json:"urls"`
	// ScrapeMethod: The method to scrape the webpage. It can be "http" or "chromedp".
	ScrapeMethod string `json:"scrape-method"`
	// IncludeHTML: Whether to include the HTML content of the webpage.
	IncludeHTML bool `json:"include-html"`
	// OnlyMainContent: Whether to scrape only the main content of the webpage.
	OnlyMainContent bool `json:"only-main-content"`
	// RemoveTags: The list of tags to remove from the HTML content.
	RemoveTags []string `json:"remove-tags,omitempty"`
	// OnlyIncludeTags: The list of tags to include in the HTML content.
	OnlyIncludeTags []string `json:"only-include-tags,omitempty"`
	// Timeout: The number of milliseconds to wait before scraping the web page. Min 0, Max 60000.
	Timeout int `json:"timeout,omitempty"`
}

// ScrapeWebpagesOutput defines the output of the scrape webpage task
type ScrapeWebpagesOutput struct {
	Pages []ScrapedPage `json:"pages"`
}

// ScrapedPage defines the struct of a webpage.
type ScrapedPage struct {
	// Content: The plain text content of the webpage.
	Content string `json:"content"`
	// Markdown: The markdown content of the webpage.
	Markdown string `json:"markdown"`
	// HTML: The HTML content of the webpage.
	HTML string `json:"html"`
	// Metadata: The metadata of the webpage.
	Metadata Metadata `json:"metadata"`
	// LinksOnPage: The list of links on the webpage.
	LinksOnPage []string `json:"links-on-page"`
}

// Metadata defines the metadata of the webpage
type Metadata struct {
	// Title: The title of the webpage.
	Title string `json:"title"`
	// Description: The description of the webpage.
	Description string `json:"description,omitempty"`
	// SourceURL: The source URL of the webpage.
	SourceURL string `json:"source-url"`
}

// ScrapeWebpages scrapes the objects of webpages
func (e *execution) ScrapeWebpages(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := ScrapeWebpagesInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("error converting input to struct: %v", err)
	}

	output := ScrapeWebpagesOutput{}

	docs, err := e.getDocsAfterRequestURLs(inputStruct.URLs, inputStruct.Timeout, inputStruct.ScrapeMethod)

	if err != nil {
		return nil, fmt.Errorf("error getting HTML page doc: %v", err)
	}

	for i, doc := range docs {
		html := getRemovedTagsHTML(doc, inputStruct)

		err = setOutput(&output, inputStruct, doc, html, i)

	}

	if err != nil {
		return nil, fmt.Errorf("error setting output: %v", err)
	}

	return base.ConvertToStructpb(output)

}

func getDocAfterRequestURL(urls []string, timeout int, scrapeMethod string) ([]*goquery.Document, error) {
	var wg sync.WaitGroup
	docCh := make(chan *goquery.Document, len(urls))
	errCh := make(chan error, len(urls))

	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			var doc *goquery.Document
			var err error
			if scrapeMethod == "http" {
				doc, err = httpRequest(url)
			} else {
				doc, err = requestToWebpage(url, timeout)
			}
			if err != nil {
				errCh <- err
				return
			}
			docCh <- doc
		}(url)
	}

	go func() {
		wg.Wait()
		close(docCh)
		close(errCh)
	}()

	docs := []*goquery.Document{}
	for doc := range docCh {
		docs = append(docs, doc)
	}

	if len(errCh) > 0 {
		return docs, <-errCh
	}

	return docs, nil

}

func httpRequest(url string) (*goquery.Document, error) {
	client := &http.Client{}
	res, err := client.Get(url)
	if err != nil {
		log.Printf("failed to make request to %s: %v", url, err)
		return nil, err
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML from %s: %v", url, err)
	}

	return doc, nil
}

func requestToWebpage(url string, timeout int) (*goquery.Document, error) {

	ctx, cancelBrowser := chromedp.NewContext(context.Background())
	defer cancelBrowser()

	var htmlContent string

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
		// Temporary solution for dynamic content.
		// There are different ways to get the dynamic content.
		// Now, we default it to scroll down the page.
		scrollDown(ctx, timeout),
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		log.Println("Cannot get dynamic content, so scrape the static content only", err)
		log.Println("htmlContent: ", htmlContent)
		if htmlContent == "" {
			return httpRequest(url)
		}
	}

	htmlReader := strings.NewReader(htmlContent)

	doc, err := goquery.NewDocumentFromReader(htmlReader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML from %s: %v", url, err)
	}

	return doc, nil
}

func scrollDown(ctx context.Context, timeout int) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		// Scroll delay is the time to wait before the next scroll
		// It is usually set 500 to 1000 milliseconds.
		// We set it to 500 milliseconds as a default value for first version.
		scrollDelay := 500 * time.Millisecond

		scrollCount := 0
		// Now, we cannot find a proper way to cancel the context for chromedp.
		// So, we set the max scrolls according to the timeout users set.
		maxScrolls := timeout / int(scrollDelay.Milliseconds())

		for scrollCount < maxScrolls {
			log.Println("Scrolling down...")

			if err := chromedp.Evaluate(`window.scrollBy(0, window.innerHeight);`, nil).Do(ctx); err != nil {
				return err
			}
			scrollCount++
			time.Sleep(scrollDelay)
			if ctx.Err() != nil {
				break
			}
		}
		return nil
	})
}

func buildTags(tags []string) string {
	tagsString := ""
	for i, tag := range tags {
		tagsString += tag
		if i < len(tags)-1 {
			tagsString += ","
		}
	}
	return tagsString
}

func setOutput(output *ScrapeWebpagesOutput, input ScrapeWebpagesInput, doc *goquery.Document, html string, idx int) error {
	plain := html2text.HTML2Text(html)

	scrapedPage := ScrapedPage{}

	scrapedPage.Content = plain
	if input.IncludeHTML {
		scrapedPage.HTML = html
	}

	url := input.URLs[idx]

	markdown, err := getMarkdown(html, url)

	if err != nil {
		return fmt.Errorf("failed to get markdown: %v", err)
	}

	scrapedPage.Markdown = markdown

	title := util.ScrapeWebpageTitle(doc)
	description := util.ScrapeWebpageDescription(doc)

	metadata := Metadata{
		Title:       title,
		Description: description,
		SourceURL:   url,
	}
	scrapedPage.Metadata = metadata

	links, err := getAllLinksOnPage(doc, url)

	if err != nil {
		return fmt.Errorf("failed to get links on page: %v", err)
	}

	scrapedPage.LinksOnPage = links

	output.Pages = append(output.Pages, scrapedPage)

	return nil

}

func getMarkdown(html, url string) (string, error) {
	domain, err := util.GetDomainFromURL(url)

	if err != nil {
		return "", fmt.Errorf("error getting domain from URL: %v", err)
	}

	markdown, err := util.ScrapeWebpageHTMLToMarkdown(html, domain)

	if err != nil {
		return "", fmt.Errorf("error converting HTML to Markdown: %v", err)
	}

	return markdown, nil
}

func getAllLinksOnPage(doc *goquery.Document, url string) ([]string, error) {
	links := []string{}

	domain, err := util.GetDomainFromURL(url)

	if err != nil {
		return nil, fmt.Errorf("error getting domain from URL: %v", err)
	}

	appendedLinks := map[string]bool{}

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		link, ok := s.Attr("href")
		if ok {
			link = getValidLink(link, domain)
			if !appendedLinks[link] {
				links = append(links, link)
				appendedLinks[link] = true
			}
		}
	})

	return links, nil
}

func getValidLink(link, domain string) string {
	if strings.HasPrefix(link, "https://") || strings.HasPrefix(link, "http://") {
		return link
	} else {
		link = "https://" + domain + link
		return link
	}
}
