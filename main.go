package main

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"log"
	"time"
)

func main() {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		// Set the headless flag to false to display the browser window
		chromedp.Flag("headless", false),
	)

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	abstract := new(string)
	keywords := new(string)
	assemblyUrl := new(string)
	okAssemblyUrl := new(bool)

	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://pubmed.ncbi.nlm.nih.gov/29708484/"),
		//chromedp.Sleep(5*time.Second),
		chromedp.WaitReady("#enc-abstract"),
		chromedp.Text("#enc-abstract p", abstract),
		chromedp.Text("#enc-abstract+ p", keywords),
		chromedp.AttributeValue("#related-links li:nth-child(1) a", "href", assemblyUrl, okAssemblyUrl),
		); err != nil {
		log.Fatal(err)
	}

	if !*okAssemblyUrl {
		log.Fatal("url do assembly nao encontrada")
	}

	if err := chromedp.Run(ctx,
		chromedp.Navigate(*assemblyUrl),
		chromedp.Sleep(5*time.Second),
	); err != nil {
		log.Fatal(err)
	}

	fmt.Println(*abstract)
	fmt.Println(*keywords)
	fmt.Println(*assemblyUrl)
}