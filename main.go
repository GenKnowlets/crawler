package main

import (
	"encoding/json"
	"fmt"
	"github.com/gocolly/colly/v2"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

type Report struct {
	OrganismName string `json:"organismName"`
	InfraspecificName string `json:"infraspecificName"`
	BioSampleUrl string `json:"bioSampleUrl"`
	Submitter string `json:"submitter"`
	Date string `json:"date"`
	FTPUrl string `json:"ftpUrl"`
	GBFFUrl string `json:"gbffUrl"`
}

type Link struct {
	Url string `json:"url"`
	Report Report `json:"report"`
}

type Assembly struct {
	Url string `json:"url"`
	Links []Link `json:"links"`
}

type Data struct {
	Abstract string `json:"abstract"`
	Keywords []string `json:"keywords"`
	Assembly Assembly `json:"assembly"`
}

func main() {
	var data Data
	c := colly.NewCollector(colly.MaxBodySize(100 * 1024 * 1024))

	c.OnHTML("#enc-abstract p", func(e *colly.HTMLElement) {
		data.Abstract = strings.TrimSpace(e.Text)
	})

	c.OnHTML("#enc-abstract+ p", func(e *colly.HTMLElement) {
		data.Keywords = strings.Split(strings.TrimSpace(strings.Replace(e.Text, "Keywords:", "", -1)), "; ")
	})

	c.OnHTML("#related-links li", func(e *colly.HTMLElement) {
		// Print link
		e.ForEachWithBreak("a", func(index int, f *colly.HTMLElement) bool{
			if strings.TrimSpace(e.Text) == "Assembly" {
				data.Assembly.Url = f.Attr("href")
				return false
			}
			return true
		})
	})

	// Start scraping
	if err := c.Visit("https://pubmed.ncbi.nlm.nih.gov/29708484/"); err != nil {
		log.Fatal(err)
	}

	if data.Assembly.Url == "" {
		log.Println("No assembly url")
		return
	}

	c.OnHTML(".rslt .title", func(e *colly.HTMLElement) {
		// Print link
		e.ForEach("a", func(index int, f *colly.HTMLElement) {
			link := Link{
				Url:    e.Request.AbsoluteURL(f.Attr("href")),
				Report: Report{},
			}
			data.Assembly.Links = append(data.Assembly.Links, link)
		})
	})

	if err := c.Visit(data.Assembly.Url); err != nil {
		log.Fatal(err)
	}

	for i, link := range data.Assembly.Links {
		c.OnHTML("dl", func(e *colly.HTMLElement) {
			var infos []string
			e.ForEach("dt", func(_ int, f *colly.HTMLElement) {
				infos = append(infos, f.Text)
			})
			e.ForEach("dd", func(index int, f *colly.HTMLElement) {
				switch infos[index] {
				case "Organism name: ":
					data.Assembly.Links[i].Report.OrganismName = f.Text
				case "Infraspecific name: ":
					data.Assembly.Links[i].Report.InfraspecificName = f.Text
				case "BioSample: ":
					url, _ := f.DOM.Find("a").Attr("href")
					data.Assembly.Links[i].Report.BioSampleUrl = e.Request.AbsoluteURL(url)
				case "Submitter: ":
					data.Assembly.Links[i].Report.Submitter = f.Text
				case "Date: ":
					data.Assembly.Links[i].Report.Date = f.Text
				}
			})
		})

		c.OnHTML(".portlet_content ul", func(g *colly.HTMLElement) {
			g.ForEachWithBreak("a",  func(_ int, f *colly.HTMLElement) bool{
				if strings.Contains(f.Text, "FTP directory") {
					data.Assembly.Links[i].Report.FTPUrl = strings.Replace(f.Attr("href"), "ftp://", "http://", 1)
					return false
				}
				return true
			})
		})


		if err := c.Visit(link.Url); err != nil {
			log.Fatal(err)
		}
	}

	for i, link := range data.Assembly.Links {
		if link.Report.FTPUrl == "" {
			continue
		}

		c.OnHTML("pre", func(e *colly.HTMLElement) {
			e.ForEachWithBreak("a",  func(index int, f *colly.HTMLElement) bool{
				if strings.Contains(f.Text, "genomic.gbff.gz") {
					data.Assembly.Links[i].Report.GBFFUrl = e.Request.AbsoluteURL(f.Attr("href"))
					return false
				}
				return true
			})
		})
		if err := c.Visit(link.Report.FTPUrl); err != nil {
			log.Fatal(err)
		}

		c.SetRequestTimeout(600*time.Second)
		c.OnResponse(func(r *colly.Response) {
			if err := r.Save(fmt.Sprintf("%v.%s", i, "gbff")); err != nil {
				log.Println("Save error:", err)
			}
		})

		start := time.Now()
		if err := c.Visit(data.Assembly.Links[i].Report.GBFFUrl); err != nil {
			log.Fatal(err)
		}
		elapsed := time.Since(start)
		log.Printf("Save file took %s", elapsed)
	}

	b, err := json.MarshalIndent(data,""," ")
	if err == nil {
		s := string(b)
		fmt.Println(s)
	}

	file, _ := json.MarshalIndent(data, "", "  ")

	_ = ioutil.WriteFile("data.json", file, 0644)
}