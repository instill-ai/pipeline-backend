package web

import (
	"fmt"
	"log"
	"regexp"

	"github.com/PuerkitoBio/goquery"
)

// Now, we decide that we do not make crawler to have the ability to scrape the website.
// However, the decision could be reverted when needed.
// So, I remain this interface design here for the future usage.
type scrapeInput interface {
	onlyMainContent() bool
	removeTags() []string
	onlyIncludeTags() []string
}

func (i ScrapeWebpagesInput) onlyMainContent() bool {
	return i.OnlyMainContent
}

func (i ScrapeWebpagesInput) removeTags() []string {
	return i.RemoveTags
}

func (i ScrapeWebpagesInput) onlyIncludeTags() []string {
	return i.OnlyIncludeTags
}

func getRemovedTagsHTML[T scrapeInput](doc *goquery.Document, input T) string {
	if input.onlyMainContent() {
		removeSelectors := []string{"header", "nav", "footer"}
		for _, selector := range removeSelectors {
			doc.Find(selector).Remove()
		}
	}

	if input.removeTags() != nil || len(input.removeTags()) > 0 {
		for _, tag := range input.removeTags() {
			doc.Find(tag).Remove()
		}
	}

	if len(input.onlyIncludeTags()) == 0 {
		html, err := doc.Html()
		if err != nil {
			log.Println("error getting HTML: ", err)
			return ""
		}
		return html
	}

	combinedHTML := ""

	tags := buildTags(input.onlyIncludeTags())
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

// targetLink filters the URL based on the filter
func targetLink(link string, filter Filter) bool {

	if len(filter.ExcludePatterns) == 0 && len(filter.IncludePatterns) == 0 {
		return true
	}

	for _, pattern := range filter.ExcludePatterns {
		if match, _ := regexp.MatchString(pattern, link); match {
			return false
		}
	}

	if len(filter.IncludePatterns) == 0 {
		return true
	}

	for _, pattern := range filter.IncludePatterns {
		if match, _ := regexp.MatchString(pattern, link); match {
			return true
		}
	}

	return false
}
