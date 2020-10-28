package main

import (
	"biocrawler/model"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gocolly/colly/v2"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func init() {
	log.SetLevel(log.InfoLevel)
}

func main() {
	var url string
	var quiet bool
	var output bool

	app := &cli.App{
		Name:  "biocrawler",
		Usage: "Crawler pubmed",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "url",
				Aliases:     []string{"u"},
				Value:       "https://pubmed.ncbi.nlm.nih.gov/29708484/",
				Usage:       "url from pubsub",
				Destination: &url,
			},
			&cli.BoolFlag{
				Name:        "quite",
				Aliases:     []string{"q"},
				Usage:       "Suppress log",
				Destination: &quiet,
			},
			&cli.BoolFlag{
				Name:        "print",
				Aliases:     []string{"p"},
				Usage:       "Display json output",
				Destination: &output,
			},
		},
		Action: func(c *cli.Context) error {
			if quiet {
				log.SetLevel(log.ErrorLevel)
			}
			crawlMain(url, output)
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func crawlMain(url string, output bool) {
	data := new(model.Data)
	c := colly.NewCollector(colly.MaxBodySize(100 * 1024 * 1024))
	c.AllowURLRevisit = true

	if err := crawlPubmed(c, data, url); err != nil {
		log.Fatal(err)
	}

	if err := crawlAssemblySearch(c, data); err != nil {
		log.Fatal(err)
	}

	for _, link := range data.Assembly.Links {
		if err := crawlAssembly(c, link.Url, link); err != nil {
			log.Fatal(err)
		}

		if link.Report.BioSample.Url != "" {
			if err := crawlBioSample(c, link.Report.BioSample.Url, link); err != nil {
				log.Fatal(err)
			}
		}

		if link.Report.FTPUrl != "" {
			if err := crawlFTPAndDownload(c, link); err != nil {
				log.Fatal(err)
			}
		}
	}

	if output {
		prettyPrintJSON(data)
	}

	file, _ := json.MarshalIndent(data, "", "  ")

	if err := ioutil.WriteFile("data.json", file, 0644); err != nil {
		log.Error(err)
	}
}

func crawlPubmed(c *colly.Collector, data *model.Data, url string) error {
	log.Infof("Crawl Pudmed %s", url)
	// Get abstract
	c.OnHTML("#enc-abstract p", func(e *colly.HTMLElement) {
		data.Abstract = strings.TrimSpace(e.Text)
	})

	// Get keywords
	c.OnHTML("#enc-abstract+ p", func(e *colly.HTMLElement) {
		data.Keywords = strings.Split(strings.TrimSpace(strings.Replace(e.Text, "Keywords:", "", -1)), "; ")
	})

	// Get DOI
	c.OnHTML(".doi .id-link", func(e *colly.HTMLElement) {
		data.DOI = e.Attr("href")
	})

	// Get Assembly URL
	c.OnHTML("#related-links li", func(e *colly.HTMLElement) {
		e.ForEachWithBreak("a", func(index int, f *colly.HTMLElement) bool {
			if strings.TrimSpace(e.Text) == "Assembly" {
				data.Assembly.Url = f.Attr("href")
				return false
			}
			return true
		})
	})

	if err := c.Visit(url); err != nil {
		return err
	}

	if data.Assembly.Url == "" {
		log.Println("No assembly url")
		return errors.New("no assembly url found")
	}

	return nil
}

func crawlAssemblySearch(c *colly.Collector, data *model.Data) error {
	log.Infof("Crawl Assembly search")
	c.OnHTML(".rslt .title", func(e *colly.HTMLElement) {
		e.ForEach("a", func(index int, f *colly.HTMLElement) {
			link := &model.Link{
				Url: e.Request.AbsoluteURL(f.Attr("href")),
				Report: &model.Report{
					BioSample: &model.BioSample{},
				},
			}
			data.Assembly.Links = append(data.Assembly.Links, link)
		})
	})

	if err := c.Visit(data.Assembly.Url); err != nil {
		return err
	}

	return nil
}

func crawlAssembly(c *colly.Collector, url string, assembly *model.Link) error {
	log.Infof("Crawl Assembly %s", url)
	c.OnHTML("dl", func(e *colly.HTMLElement) {
		var infos []string
		e.ForEach("dt", func(_ int, f *colly.HTMLElement) {
			infos = append(infos, f.Text)
		})
		e.ForEach("dd", func(index int, f *colly.HTMLElement) {
			switch infos[index] {
			case "Organism name: ":
				assembly.Report.OrganismName = f.Text
				url, _ := f.DOM.Find("a").Attr("href")
				assembly.Report.TaxonomyUrl = e.Request.AbsoluteURL(url)
			case "Infraspecific name: ":
				assembly.Report.InfraspecificName = f.Text
			case "BioSample: ":
				url, _ := f.DOM.Find("a").Attr("href")
				assembly.Report.BioSample.Url = e.Request.AbsoluteURL(url)
			case "Submitter: ":
				assembly.Report.Submitter = f.Text
			case "Date: ":
				assembly.Report.Date = f.Text
			}
		})
	})

	c.OnHTML(".portlet_content ul", func(g *colly.HTMLElement) {
		g.ForEachWithBreak("a", func(_ int, f *colly.HTMLElement) bool {
			if strings.Contains(f.Text, "FTP directory") {
				assembly.Report.FTPUrl = strings.Replace(f.Attr("href"), "ftp://", "http://", 1)
				return false
			}
			return true
		})
	})

	c.OnHTML(".portlet_content ul", func(g *colly.HTMLElement) {
		g.ForEachWithBreak("a", func(_ int, f *colly.HTMLElement) bool {
			if strings.Contains(f.Text, "FTP directory") {
				assembly.Report.FTPUrl = strings.Replace(f.Attr("href"), "ftp://", "http://", 1)
				return false
			}
			return true
		})
	})

	if err := c.Visit(url); err != nil {
		return err
	}

	return nil
}

func crawlBioSample(c *colly.Collector, url string, assembly *model.Link) error {
	log.Infof("Crawl BioSample %s", url)

	c.OnHTML("tbody", func(table *colly.HTMLElement) {
		table.ForEach("tr", func(index int, row *colly.HTMLElement) {
			switch row.DOM.Find("th").Text() {
			case "strain":
				assembly.Report.BioSample.Strain = strings.TrimSpace(row.DOM.Find("td").Text())
			case "collection date":
				assembly.Report.BioSample.CollectionDate = strings.TrimSpace(row.DOM.Find("td").Text())
			case "broad-scale environmental context":
				assembly.Report.BioSample.BroadScaleEnvironmentalContext = strings.TrimSpace(row.DOM.Find("td").Text())
			case "local-scale environmental context":
				assembly.Report.BioSample.LocalScaleEnvironmentalContext = strings.TrimSpace(row.DOM.Find("td").Text())
			case "environmental medium":
				assembly.Report.BioSample.EnvironmentalMedium = strings.TrimSpace(row.DOM.Find("td").Text())
			case "geographic location":
				assembly.Report.BioSample.GeographicLocation = strings.TrimSpace(row.DOM.Find("td").Text())
			case "latitude and longitude":
				assembly.Report.BioSample.LatLong = strings.TrimSpace(row.DOM.Find("td").Text())
			case "host":
				assembly.Report.BioSample.Host = strings.TrimSpace(row.DOM.Find("td").Text())
			case "isolation and growth condition":
				assembly.Report.BioSample.IsolationAndGrowthCondition = strings.TrimSpace(row.DOM.Find("td").Text())
			case "number of replicons":
				assembly.Report.BioSample.NumberOfReplicons = strings.TrimSpace(row.DOM.Find("td").Text())
			case "ploidy":
				assembly.Report.BioSample.Ploidy = strings.TrimSpace(row.DOM.Find("td").Text())
			case "propagation":
				assembly.Report.BioSample.Propagation = strings.TrimSpace(row.DOM.Find("td").Text())
			}
		})
	})

	if err := c.Visit(url); err != nil {
		return err
	}

	return nil
}

func crawlFTPAndDownload(c *colly.Collector, assembly *model.Link) error {
	log.Infof("Crawl FTP %s", assembly.Url)
	c.OnHTML("pre", func(e *colly.HTMLElement) {
		e.ForEachWithBreak("a", func(index int, f *colly.HTMLElement) bool {
			if strings.Contains(f.Text, "genomic.gbff.gz") {
				assembly.Report.GBFFUrl = e.Request.AbsoluteURL(f.Attr("href"))
				return false
			}
			return true
		})
	})
	if err := c.Visit(assembly.Report.FTPUrl); err != nil {
		return err
	}

	if assembly.Report.GBFFUrl != "" {
		saveGBFF(c, strings.Replace(strings.Split(assembly.Report.GBFFUrl, "/")[len(strings.Split(assembly.Report.GBFFUrl, "/"))-1], ".gbff.gz", "", 1), assembly.Report.GBFFUrl)
	}

	return nil
}

func saveGBFF(c *colly.Collector, name, url string) {
	filename := fmt.Sprintf("%v.%s", name, "gbff")

	c.SetRequestTimeout(600 * time.Second)
	c.OnResponse(func(r *colly.Response) {
		if err := r.Save(filename); err != nil {
			log.Error("Save error:", err)
		}
	})

	start := time.Now()
	if err := c.Visit(url); err != nil {
		log.Fatal(err)
	}
	elapsed := time.Since(start)
	log.Infof("Save file %s took %s", filename, elapsed)
}

func prettyPrintJSON(data *model.Data) {
	b, err := json.MarshalIndent(data, "", " ")
	if err == nil {
		s := string(b)
		fmt.Println(s)
	}
}
