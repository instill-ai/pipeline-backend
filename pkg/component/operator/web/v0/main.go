//go:generate compogen readme ./config ./README.mdx --extraContents TASK_SCRAPE_WEBPAGE=.compogen/scrape_webpage.mdx --extraContents bottom=.compogen/bottom.mdx
package web

import (
	"context"
	"fmt"
	"io"
	"sync"

	_ "embed"

	"github.com/PuerkitoBio/goquery"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	taskCrawlWebsite  = "TASK_CRAWL_SITE"
	taskScrapeWebpage = "TASK_SCRAPE_PAGE"
	taskScrapeSitemap = "TASK_SCRAPE_SITEMAP"
)

var (
	//go:embed config/definition.json
	definitionJSON []byte
	//go:embed config/tasks.json
	tasksJSON []byte

	once sync.Once
	comp *component
)

type component struct {
	base.Component
}

type execution struct {
	base.ComponentExecution
	execute               func(*structpb.Struct) (*structpb.Struct, error)
	externalCaller        func(url string) (ioCloser io.ReadCloser, err error)
	getDocAfterRequestURL func(url string, timeout int) (*goquery.Document, error)
}

func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionJSON, nil, tasksJSON, nil)
		if err != nil {
			panic(err)
		}
	})
	return comp
}

func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	e := &execution{
		ComponentExecution: x,
	}

	switch x.Task {
	case taskCrawlWebsite:
		e.execute = e.CrawlWebsite
	case taskScrapeSitemap:
		// To make mocking easier
		e.externalCaller = scrapSitemapCaller
		e.execute = e.ScrapeSitemap
	case taskScrapeWebpage:
		e.getDocAfterRequestURL = getDocAfterRequestURL
		e.execute = e.ScrapeWebpage
	default:
		return nil, fmt.Errorf("%s task is not supported", x.Task)
	}

	return e, nil
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.SequentialExecutor(ctx, jobs, e.execute)
}
