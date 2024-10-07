package main

import (
	"errors"
	"flag"
	"fmt"
	"log"

	"ciascrape/pkg/anythingllm"
	"ciascrape/pkg/cia"
)

const (
	defaultMaxPages = 50
)

var (
	ErrInvalidConfig = errors.New("invalid config")
)

type Config struct {
	Collection  string
	MaxPages    int
	AnythingLLM *anythingllm.Config
}

func NewConfig(collection string) *Config {
	return &Config{
		Collection:  collection,
		MaxPages:    defaultMaxPages,
		AnythingLLM: anythingllm.NewConfig(),
	}
}

func (c *Config) WithMaxPages(maxPages int) *Config {
	c.MaxPages = maxPages
	return c
}

func (c *Config) WithAnythingLLM(config *anythingllm.Config) *Config {
	c.AnythingLLM = config
	return c
}

func ConfigFromFlags() *Config {
	maxPages := flag.Int("pages", defaultMaxPages, "Maximum number of pages to scrape")
	collection := flag.String("collection", "", "Collection to scrape")
	aEndpoint := flag.String("anythingllm-endpoint", anythingllm.DefaultEndpoint, "AnythingLLM endpoint")
	aKey := flag.String("anythingllm-key", "", "AnythingLLM key")

	flag.Parse()

	anythingLLM := anythingllm.NewConfig().
		WithEndpoint(*aEndpoint).WithAPIKey(*aKey)

	if *collection == "" {
		log.Fatal("Collection is required")
	}

	return NewConfig(*collection).WithAnythingLLM(anythingLLM).WithMaxPages(*maxPages)
}

func (c *Config) Validate() error {
	if c.Collection == "" {
		return fmt.Errorf("%w: missing collection name", ErrInvalidConfig)
	}
	ciaCol := cia.NewCollection(c.Collection)
	if err := ciaCol.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	if c.MaxPages <= 0 {
		return fmt.Errorf("%w: max pages must be positive", ErrInvalidConfig)
	}
	if err := c.AnythingLLM.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	return nil
}
