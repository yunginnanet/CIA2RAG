package main

import (
	"context"
	"log"

	"github.com/l0nax/go-spew/spew"

	"ciascrape/pkg/cia"
)

func run(cfg *Config) error {
	ciaCol := cia.NewCollection(cfg.Collection).WithMaxPages(cfg.MaxPages)

	go func() {
		if err := ciaCol.GetPages(); err != nil {
			log.Printf("[err] failed to get pages: %v", err)
		}
	}()

	pages := ciaCol.Drain(context.Background())

	count := 0

	for page := range pages {
		log.Printf("uploading page: %s", page)
		doc, err := cfg.AnythingLLM.UploadLink(page)
		if err != nil {
			log.Printf("[err] failed to upload link: %v", err)
			continue
		}
		spew.Dump(doc)
		count++
	}

	log.Printf("uploaded %d links", count)

	return nil
}

func main() {
	cfg := ConfigFromFlags()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}
	log.Printf("configuration validated: %v", cfg)
	if err := run(cfg); err != nil {
		log.Fatalf("run failed: %v", err)
	}
}
