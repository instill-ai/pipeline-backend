package web

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/exec"
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
		return getStaticContent(url)
	}

}

func getStaticContent(url string) (*goquery.Document, error) {

	doc, _ := httpRequest(url)

	if doc != nil {
		return doc, nil
	}

	// TODO: Investigate the root cause of the handshake error and remove this temporary solution.
	// We got handshake error when using http request, so we use curl request instead.
	doc, err := curlRequest(url)

	if err != nil {
		return nil, fmt.Errorf("error getting static content: %v", err)
	}

	return doc, nil
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

// TODO: Investigate the root cause of the handshake error and remove this temporary solution.
// We got handshake error when using http request, so we use curl request instead.
func curlRequest(url string) (*goquery.Document, error) {
	cmd := exec.Command("curl", "-s", url)

	output, err := cmd.Output()

	if err != nil {
		return nil, fmt.Errorf("error running curl command: %v", err)
	}

	htmlReader := strings.NewReader(string(output))

	doc, err := goquery.NewDocumentFromReader(htmlReader)

	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML from %s: %v", url, err)
	}

	return doc, nil
}

func requestToWebpage(url string, timeout int) (*goquery.Document, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer cancel()

	ctx, cancelBrowser := chromedp.NewContext(ctx)
	defer cancelBrowser()

	var htmlContent string

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
		chromedp.OuterHTML("html", &htmlContent),
	)
	if err != nil {
		log.Println("Cannot get dynamic content, so scrape the static content only", err)
		return getStaticContent(url)
	}

	htmlReader := strings.NewReader(htmlContent)

	doc, err := goquery.NewDocumentFromReader(htmlReader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML from %s: %v", url, err)
	}

	return doc, nil
}

func getRemovedTagsHTML(doc *goquery.Document, input ScrapeWebpageInput) string {
	if input.OnlyMainContent {
		removeSelectors := []string{"header", "nav", "footer"}
		for _, selector := range removeSelectors {
			doc.Find(selector).Remove()
		}
	}

	if input.RemoveTags != nil || len(input.RemoveTags) > 0 {
		for _, tag := range input.RemoveTags {
			doc.Find(tag).Remove()
		}
	}

	if len(input.OnlyIncludeTags) == 0 {
		html, err := doc.Html()
		if err != nil {
			log.Println("error getting HTML: ", err)
			return ""
		}
		return html
	}

	combinedHTML := ""

	tags := buildTags(input.OnlyIncludeTags)
	doc.Find(tags).Each(func(i int, s *goquery.Selection) {
		html, err := s.Html()
		if err != nil {
			log.Println("error getting HTML: ", err)
			combinedHTML += "\n"
		}
		combinedHTML += fmt.Sprintf("<%s>%s</%s>\n", s.Nodes[0].Data, html, s.Nodes[0].Data)
	})

	return combinedHTML
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
