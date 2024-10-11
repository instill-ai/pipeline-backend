package web

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/k3a/html2text"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

type ScrapeWebpageInput struct {
	URL             string   `json:"url"`
	IncludeHTML     bool     `json:"include-html"`
	OnlyMainContent bool     `json:"only-main-content"`
	RemoveTags      []string `json:"remove-tags,omitempty"`
	OnlyIncludeTags []string `json:"only-include-tags,omitempty"`
	Timeout         int      `json:"timeout,omitempty"`
}

type ScrapeWebpageOutput struct {
	Content     string   `json:"content"`
	Markdown    string   `json:"markdown"`
	HTML        string   `json:"html"`
	Metadata    Metadata `json:"metadata"`
	LinksOnPage []string `json:"links-on-page"`
}

type Metadata struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	SourceURL   string `json:"source-url"`
}

func (e *execution) ScrapeWebpage(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := ScrapeWebpageInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("error converting input to struct: %v", err)
	}

	output := ScrapeWebpageOutput{}

	doc, err := e.getDocAfterRequestURL(inputStruct.URL, inputStruct.Timeout)

	if err != nil {
		return nil, fmt.Errorf("error getting HTML page doc: %v", err)
	}

	html := getRemovedTagsHTML(doc, inputStruct)

	err = setOutput(&output, inputStruct, doc, html)

	if err != nil {
		return nil, fmt.Errorf("error setting output: %v", err)
	}

	return base.ConvertToStructpb(output)

}

func getDocAfterRequestURL(url string, timeout int) (*goquery.Document, error) {

	if timeout > 0 {
		return requestToWebpage(url, timeout)
	} else {
		return httpRequest(url)
	}

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

func setOutput(output *ScrapeWebpageOutput, input ScrapeWebpageInput, doc *goquery.Document, html string) error {
	plain := html2text.HTML2Text(html)

	output.Content = plain
	if input.IncludeHTML {
		output.HTML = html
	}

	markdown, err := getMarkdown(html, input.URL)

	if err != nil {
		return fmt.Errorf("failed to get markdown: %v", err)
	}

	output.Markdown = markdown

	title := util.ScrapeWebpageTitle(doc)
	description := util.ScrapeWebpageDescription(doc)

	metadata := Metadata{
		Title:       title,
		Description: description,
		SourceURL:   input.URL,
	}
	output.Metadata = metadata

	links, err := getAllLinksOnPage(doc, input.URL)

	if err != nil {
		return fmt.Errorf("failed to get links on page: %v", err)
	}

	output.LinksOnPage = links

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
