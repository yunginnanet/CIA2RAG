package main

import "log"

func run(cfg *Config) error {
	return nil
}

func main() {
	cfg := ConfigFromFlags()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}
	// cfg.Run()
}
