package main

import (
	"biocrawler/model"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/gocolly/colly/v2"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
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
	var downloadGBFF bool
	var lookForRelated bool

	app := &cli.App{
		Name:  "biocrawler",
		Usage: "Crawler",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "url",
				Aliases:     []string{"u"},
				Value:       "https://pubmed.ncbi.nlm.nih.gov/29708484/",
				Usage:       "url from pubsub",
				Destination: &url,
			},
			&cli.BoolFlag{
				Name:        "quiet",
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
			&cli.BoolFlag{
				Name:        "gbff",
				Aliases:     []string{"g"},
				Usage:       "Download GBFF file",
				Destination: &downloadGBFF,
			},
			&cli.BoolFlag{
				Name:        "related",
				Aliases:     []string{"r"},
				Usage:       "Look for related assemblies",
				Destination: &lookForRelated,
			},
		},
		Action: func(c *cli.Context) error {
			if quiet {
				log.SetLevel(log.ErrorLevel)
			}
			if strings.Contains(url, "pubmed.ncbi.nlm.nih.gov") {
				crawlMain(url, output, downloadGBFF, lookForRelated, true)
			} else if strings.Contains(url, "ncbi.nlm.nih.gov/assembly") {
				crawlMain(url, output, downloadGBFF, lookForRelated, false)
			} else {
				crawlMain(url, output, downloadGBFF, lookForRelated, true)
			}

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func crawlMain(url string, output, downloadGBFF, lookForRelated bool, fromPubmed bool) {
	data := new(model.Data)
	c := colly.NewCollector(colly.MaxBodySize(100 * 1024 * 1024))
	c.AllowURLRevisit = true
	moreAssembliesUrl := ""

	if fromPubmed {
		if err := crawlPubmed(c, data, url, fromPubmed); err != nil {
			log.Fatal(err)
		}

		if err := crawlAssemblySearch(c, data); err != nil {
			log.Fatal(err)
		}
	} else {
		data.Assembly.Links = []model.Link{{
			Url: url,
		}}
	}

	for i := range data.Assembly.Links {
		pubmedUrl, otherAssembliesUrl, report, err := crawlAssembly(c, fromPubmed, data.Assembly.Links[i].Url)
		if err != nil {
			log.Fatal(err)
		}
		data.Assembly.Links[i].Report = report

		if !fromPubmed {
			if err := crawlPubmed(c, data, pubmedUrl, fromPubmed); err != nil {
				log.Fatal(err)
			}
		}

		if data.Assembly.Links[i].Report.BioSample.Url != "" {
			bioSample, err := crawlBioSample(c, data.Assembly.Links[i].Report.BioSample.Url)
			if err != nil {
				log.Fatal(err)
			}
			data.Assembly.Links[i].Report.BioSample = bioSample
		}

		if data.Assembly.Links[i].Report.FTPUrl != "" {
			if err := crawlFTPAndDownload(c, data.Assembly.Links[i], downloadGBFF); err != nil {
				log.Fatal(err)
			}
		}

		moreAssembliesUrl = otherAssembliesUrl
	}
	originalSize := len(data.Assembly.Links)

	if moreAssembliesUrl != "" && lookForRelated{
		var existingAssemblies []string
		for i := range data.Assembly.Links {
			existingAssemblies = append(existingAssemblies, data.Assembly.Links[i].Url)
		}

		otherAssembliesUrls, err := crawlOtherAssemblies(moreAssembliesUrl)
		if err != nil {
			log.Fatal(err)
		}

		for i := range otherAssembliesUrls {
			if contains(existingAssemblies, otherAssembliesUrls[i]) {
				continue
			}

			link := model.Link{
				Url: otherAssembliesUrls[i],
				Report: model.Report{
					BioSample: model.BioSample{},
				},
			}

			data.Assembly.Links = append(data.Assembly.Links, link)
		}

		//Crawl a maximum of 100 other assemblies if found
		for i := range data.Assembly.Links {
			if i < originalSize {
				continue
			}
			_, _, report, err := crawlAssembly(c, true, data.Assembly.Links[i].Url)
			if err != nil {
				log.Fatal(err)
			}

			data.Assembly.Links[i].Report = report

			if data.Assembly.Links[i].Report.BioSample.Url != "" {
				bioSample, err := crawlBioSample(c, data.Assembly.Links[i].Report.BioSample.Url)
				if err != nil {
					log.Fatal(err)
				}
				data.Assembly.Links[i].Report.BioSample = bioSample
			}

			if data.Assembly.Links[i].Report.FTPUrl != "" {
				if err := crawlFTPAndDownload(c, data.Assembly.Links[i], downloadGBFF); err != nil {
					log.Fatal(err)
				}
			}
		}
	}

	prettyJson := indentJson(data)
	if output {
		fmt.Println(string(prettyJson))
	}

	if err := ioutil.WriteFile("./input/data.json", prettyJson, 0644); err != nil {
		log.Error(err)
	}

	cmd := exec.Command("python3", "src/main.py")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if err != nil {
		fmt.Println(err)
		return
	}
}

func crawlOtherAssemblies(url string) ([]string, error) {
	var otherAssembliesUrls []string
	log.Infof("Crawl other Assemblies found")

	client := resty.New()
	resp, err := client.R().
		Get(strings.Replace(url, "organism/", "organism/browse-organism/", 1) + "?p$l=Ajax&pageSize=100")
	if err != nil {
		return nil, err
	}

	result := string(resp.Body())
	r := regexp.MustCompile(`<a href="(.*)"`)
	matches := r.FindAllStringSubmatch(result, -1)
	for _, v := range matches {
		otherAssembliesUrls = append(otherAssembliesUrls, v[1])
	}

	return otherAssembliesUrls, nil
}

func crawlPubmed(c *colly.Collector, data *model.Data, url string, fromPubmed bool) error {
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
	if fromPubmed {
		c.OnHTML("#related-links li", func(e *colly.HTMLElement) {
			e.ForEachWithBreak("a", func(index int, f *colly.HTMLElement) bool {
				if strings.TrimSpace(e.Text) == "Assembly" {
					data.Assembly.Url = f.Attr("href")
					return false
				}
				return true
			})
		})
	}

	if err := c.Visit(url); err != nil {
		return err
	}

	if fromPubmed && data.Assembly.Url == "" {
		log.Println("No assembly url")
		return errors.New("no assembly url found")
	}

	return nil
}

func crawlAssemblySearch(c *colly.Collector, data *model.Data) error {
	log.Infof("Crawl Assembly search")
	c.OnHTML(".rslt .title", func(e *colly.HTMLElement) {
		e.ForEach("a", func(index int, f *colly.HTMLElement) {
			report := model.Report{}
			sample := model.BioSample{}
			report.BioSample = sample
			link := model.Link{
				Url: e.Request.AbsoluteURL(f.Attr("href")),
			}
			data.Assembly.Links = append(data.Assembly.Links, link)
		})
	})

	if err := c.Visit(data.Assembly.Url); err != nil {
		return err
	}

	return nil
}

func crawlAssembly(c *colly.Collector, fromPubmed bool, url string) (string, string, model.Report, error) {
	report := model.Report{}
	pubmedUrl := ""
	otherAssembliesUrl := ""
	log.Infof("Crawl Assembly %s", url)
	c.OnHTML("dl", func(e *colly.HTMLElement) {
		var infos []string
		e.ForEach("dt", func(_ int, f *colly.HTMLElement) {
			infos = append(infos, f.Text)
		})
		e.ForEach("dd", func(i int, f *colly.HTMLElement) {
			switch infos[i] {
			case "Organism name: ":
				report.OrganismName = f.Text
				url, _ := f.DOM.Find("a").Attr("href")
				report.TaxonomyUrl = e.Request.AbsoluteURL(url)
			case "Infraspecific name: ":
				report.InfraspecificName = f.Text
			case "BioSample: ":
				url, _ := f.DOM.Find("a").Attr("href")
				report.BioSample.Url = e.Request.AbsoluteURL(url)
			case "Submitter: ":
				report.Submitter = f.Text
			case "Date: ":
				report.Date = f.Text
			}
		})
	})

	c.OnHTML(".portlet_content ul", func(g *colly.HTMLElement) {
		g.ForEachWithBreak("a", func(_ int, f *colly.HTMLElement) bool {
			if strings.Contains(f.Text, "FTP directory") {
				report.FTPUrl = strings.Replace(f.Attr("href"), "ftp://", "http://", 1)
				return false
			}
			return true
		})
	})

	c.OnHTML(".portlet_content ul", func(g *colly.HTMLElement) {
		g.ForEachWithBreak("a", func(_ int, f *colly.HTMLElement) bool {
			if strings.Contains(f.Text, "FTP directory") {
				report.FTPUrl = strings.Replace(f.Attr("href"), "ftp://", "http://", 1)
				return false
			}
			return true
		})
	})

	if !fromPubmed {
		c.OnHTML(".DiscoveryDbLinks ul", func(g *colly.HTMLElement) {
			g.ForEachWithBreak("li", func(_ int, f *colly.HTMLElement) bool {
				if strings.Contains(f.DOM.Find("a").Text(), "PubMed") {
					url, _ := f.DOM.Find("a").Attr("href")
					pubmedUrl =  g.Request.AbsoluteURL(url)
					return false
				}
				return true
			})
		})
	}

	c.OnHTML(".more_genome_data-cont", func(g *colly.HTMLElement) {
		g.ForEachWithBreak(".more_genome_data", func(_ int, f *colly.HTMLElement) bool {
			if strings.Contains(f.DOM.Find("h3").Text(), "assemblies for this organism") {
				href, _ := f.DOM.Find("a").Attr("href")
				if href != "" {
					otherAssembliesUrl = f.Request.AbsoluteURL(href)
				}
				return false
			}
			return true
		})
	})

	if err := c.Visit(url); err != nil {
		return "", "", model.Report{}, err
	}

	log.Info(report.InfraspecificName)
	return pubmedUrl, otherAssembliesUrl, report, nil
}

func crawlBioSample(c *colly.Collector, url string) (model.BioSample, error) {
	log.Infof("Crawl BioSample %s", url)
	bioSample := model.BioSample{Url: url}

	c.OnHTML("tbody", func(table *colly.HTMLElement) {
		table.ForEach("tr", func(index int, row *colly.HTMLElement) {
			switch row.DOM.Find("th").Text() {
			case "strain":
				bioSample.Strain = strings.TrimSpace(row.DOM.Find("td").Text())
			case "collection date":
				bioSample.CollectionDate = strings.TrimSpace(row.DOM.Find("td").Text())
			case "broad-scale environmental context":
				bioSample.BroadScaleEnvironmentalContext = strings.TrimSpace(row.DOM.Find("td").Text())
			case "local-scale environmental context":
				bioSample.LocalScaleEnvironmentalContext = strings.TrimSpace(row.DOM.Find("td").Text())
			case "environmental medium":
				bioSample.EnvironmentalMedium = strings.TrimSpace(row.DOM.Find("td").Text())
			case "geographic location":
				bioSample.GeographicLocation = strings.TrimSpace(row.DOM.Find("td").Text())
			case "latitude and longitude":
				bioSample.LatLong = strings.TrimSpace(row.DOM.Find("td").Text())
			case "host":
				bioSample.Host = strings.TrimSpace(row.DOM.Find("td").Text())
			case "isolation and growth condition":
				bioSample.IsolationAndGrowthCondition = strings.TrimSpace(row.DOM.Find("td").Text())
			case "number of replicons":
				bioSample.NumberOfReplicons = strings.TrimSpace(row.DOM.Find("td").Text())
			case "ploidy":
				bioSample.Ploidy = strings.TrimSpace(row.DOM.Find("td").Text())
			case "propagation":
				bioSample.Propagation = strings.TrimSpace(row.DOM.Find("td").Text())
			}
		})
	})

	if err := c.Visit(url); err != nil {
		return model.BioSample{}, err
	}

	return  bioSample, nil
}

func crawlFTPAndDownload(c *colly.Collector, assembly model.Link, downloadGBFF bool) error {
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

	filename := strings.Replace(strings.Split(assembly.Report.GBFFUrl, "/")[len(strings.Split(assembly.Report.GBFFUrl, "/"))-1], ".gbff.gz", "", 1)
	assembly.Report.GBFFPath = "../output/" + filename

	if downloadGBFF && assembly.Report.GBFFUrl != "" {
		saveGBFF(c, filename, assembly.Report.GBFFUrl)
	}

	return nil
}

func saveGBFF(c *colly.Collector, name, url string) {
	log.Infof("Downloading GBFF file")
	filename := fmt.Sprintf("./output/%v.%s", name, "gbff")

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
	log.Infof("Saving file %s took %s", filename, elapsed)
}

func indentJson(data *model.Data) []byte {
	bf := bytes.NewBuffer([]byte{})
	jsonEncoder := json.NewEncoder(bf)
	jsonEncoder.SetEscapeHTML(false)
	jsonEncoder.SetIndent("", "  ")
	if err := jsonEncoder.Encode(data); err != nil {
		log.Error("Json encoder error:", err)
	}
	return bf.Bytes()
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if strings.Contains(e, a) {
			return true
		}
	}
	return false
}
