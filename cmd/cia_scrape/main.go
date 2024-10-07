package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/l0nax/go-spew/spew"

	"ciascrape/pkg/anythingllm"
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
	retries := 0

	for page := range pages {
		log.Printf("uploading page: %s", page)
		doc, err := cfg.AnythingLLM.UploadLink(page)
		if errors.Is(err, anythingllm.ErrDuplicate) {
			log.Printf("duplicate link: %s", page)
			continue
		}
		if err != nil {
			if errors.Is(err, anythingllm.ErrAccessDenied) {
				retries++
				log.Printf("[err] access denied (%d), retrying and sleeping...", retries)
				if err := cfg.AnythingLLM.DeleteDocument(doc.Location); err != nil {
					log.Printf("[err] failed to delete document '%s': %v", doc.Location, err)
					return err
				}
				log.Printf("deleted document '%s'", doc.Location)
				go func() {
					time.Sleep(time.Second)
					pages <- page
				}()
				switch {
				case retries > 10 && retries <= 100:
					time.Sleep(time.Millisecond * time.Duration(retries*100))
				case retries > 100:
					time.Sleep(time.Second * time.Duration(retries))
				default:
					time.Sleep(time.Second * time.Duration(retries))
				}
				continue
			}
			log.Printf("[err] failed to upload link: %v", err)
			continue
		}
		spew.Dump(doc)
		if err := cfg.AnythingLLM.AddDocument(doc); err != nil {
			log.Printf("[err] failed to add document '%s': %v", doc.ID, err)
			return err
		}
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
