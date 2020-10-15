package main

import (
	"fmt"
	"github.com/gocolly/colly/v2"
	"log"
	"strings"
	"time"
)

func main() {
	c := colly.NewCollector(colly.MaxBodySize(100 * 1024 * 1024))

	c.OnHTML("#enc-abstract p", func(e *colly.HTMLElement) {
		fmt.Printf("Abstract found: %q\n", strings.TrimSpace(e.Text))
	})

	c.OnHTML("#enc-abstract+ p", func(e *colly.HTMLElement) {
		fmt.Printf("Abstract found: %q\n", strings.TrimSpace(e.Text))
	})

	c.OnHTML("#related-links li:nth-child(1) a", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		// Print link
		fmt.Printf("Assembly link found: %q -> %s\n", strings.TrimSpace(e.Text), link)
	})

	// Before making a request print "Visiting ..."
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	// Start scraping
	if err := c.Visit("https://pubmed.ncbi.nlm.nih.gov/29708484/"); err != nil {
		log.Fatal(err)
	}

	// Exemplo para fazer download. Tá descomprimindo automaticamente, acho que seria interessante não descomprimir caso
	// o script do python aceite em .gz
	c.SetRequestTimeout(60*time.Second)
	c.OnResponse(func(r *colly.Response) {
		if err := r.Save(fmt.Sprintf("%s.%s", "arquivo", "gbff")); err != nil {
			log.Println("Save error:", err)
		}
	})

	start := time.Now()
	if err := c.Visit("https://ftp.ncbi.nlm.nih.gov/genomes/all/GCA/003/177/105/GCA_003177105.1_ASM317710v1/GCA_003177105.1_ASM317710v1_genomic.gbff.gz"); err != nil {
		log.Fatal(err)
	}
	elapsed := time.Since(start)
	log.Printf("Save file took %s", elapsed)
}