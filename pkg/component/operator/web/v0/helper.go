package web

import (
	"fmt"
	"log"

	"github.com/PuerkitoBio/goquery"
)

type scrapeInput interface {
	onlyMainContent() bool
	removeTags() []string
	onlyIncludeTags() []string
}

func (s ScrapeWebpageInput) onlyMainContent() bool {
	return s.OnlyMainContent
}

func (s ScrapeWebpageInput) removeTags() []string {
	return s.RemoveTags
}

func (s ScrapeWebpageInput) onlyIncludeTags() []string {
	return s.OnlyIncludeTags
}

func (s CrawlWebsiteInput) onlyMainContent() bool {
	return s.OnlyMainContent
}

func (s CrawlWebsiteInput) removeTags() []string {
	return s.RemoveTags
}

func (s CrawlWebsiteInput) onlyIncludeTags() []string {
	return s.OnlyIncludeTags
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
