package web

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type ScrapeSitemapInput struct {
	URL string `json:"url"`
}

type ScrapeSitemapOutput struct {
	List []SiteInformation `json:"list"`
}

type SiteInformation struct {
	Loc string `json:"loc"`
	// Follow ISO 8601 format
	LastModifiedTime string  `json:"lastmod"`
	ChangeFrequency  string  `json:"changefreq,omitempty"`
	Priority         float64 `json:"priority,omitempty"`
}

type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	Urls    []URL    `xml:"url"`
}

type URL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod"`
	ChangeFreq string `xml:"changefreq"`
	Priority   string `xml:"priority"`
}

func (e *execution) ScrapeSitemap(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := ScrapeSitemapInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	ioCloser, err := e.externalCaller(inputStruct.URL)

	if err != nil {
		return nil, fmt.Errorf("failed to scrap the URL: %v", err)
	}

	defer ioCloser.Close()

	body, err := io.ReadAll(ioCloser)

	if err != nil {
		return nil, fmt.Errorf("failed to read the response body: %v", err)
	}

	var urlSet URLSet
	err = xml.Unmarshal(body, &urlSet)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %v", err)
	}

	list := []SiteInformation{}
	for _, url := range urlSet.Urls {
		priority, err := strconv.ParseFloat(url.Priority, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse priority: %v", err)
		}

		list = append(list, SiteInformation{
			Loc:              url.Loc,
			LastModifiedTime: url.LastMod,
			ChangeFrequency:  url.ChangeFreq,
			Priority:         priority,
		})
	}

	output := ScrapeSitemapOutput{
		List: list,
	}

	outputStruct, err := base.ConvertToStructpb(output)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}
	return outputStruct, nil
}

func scrapSitemapCaller(url string) (io.ReadCloser, error) {
	resp, err := http.Get(url)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch the URL: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error: status code %d", resp.StatusCode)
	}
	return resp.Body, nil
}
