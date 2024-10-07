package anythingllm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type Workspace struct {
	Id                   int         `json:"id"`
	Name                 string      `json:"name"`
	Slug                 string      `json:"slug"`
	VectorTag            interface{} `json:"vectorTag"`
	CreatedAt            time.Time   `json:"createdAt"`
	OpenAiTemp           float64     `json:"openAiTemp"`
	OpenAiHistory        int         `json:"openAiHistory"`
	LastUpdatedAt        time.Time   `json:"lastUpdatedAt"`
	OpenAiPrompt         string      `json:"openAiPrompt"`
	SimilarityThreshold  float64     `json:"similarityThreshold"`
	ChatProvider         interface{} `json:"chatProvider"`
	ChatModel            interface{} `json:"chatModel"`
	TopN                 int         `json:"topN"`
	ChatMode             string      `json:"chatMode"`
	PfpFilename          interface{} `json:"pfpFilename"`
	AgentProvider        interface{} `json:"agentProvider"`
	AgentModel           interface{} `json:"agentModel"`
	QueryRefusalResponse string      `json:"queryRefusalResponse"`
	Documents            []struct {
		Id            int       `json:"id"`
		DocId         string    `json:"docId"`
		Filename      string    `json:"filename"`
		Docpath       string    `json:"docpath"`
		WorkspaceId   int       `json:"workspaceId"`
		Metadata      string    `json:"metadata"`
		Pinned        bool      `json:"pinned"`
		Watched       bool      `json:"watched"`
		CreatedAt     time.Time `json:"createdAt"`
		LastUpdatedAt time.Time `json:"lastUpdatedAt"`
	} `json:"documents"`
	Threads []interface{} `json:"threads"`
}

type WorkspaceResponse struct {
	Workspaces []Workspace `json:"workspace"`
}

type UpdateEmbeddings struct {
	Adds    []string `json:"adds,omitempty"`
	Deletes []string `json:"deletes,omitempty"`
}

func (c *Config) AddDocumentItem(doc *Item) error {
	startQueueOnce.Do(c.docQueueFlush)

	name := doc.Name
	if strings.Contains(name, ".html") {
		name = strings.ReplaceAll(name, ".html", "")
	}
	if !strings.HasPrefix(name, "url-") {
		name = "url-" + name
	}

	if !strings.Contains(name, doc.ID) {
		name = strings.ReplaceAll(name, ".json", "") + "-" + doc.ID
	}

	if !strings.HasSuffix(name, ".json") {
		name = name + ".json"
	}

	if strings.Count(name, ".json") > 1 {
		name = strings.Split(name, ".json")[0] + ".json"
	}

	doc2 := &Document{
		Location: fmt.Sprintf("%s/%s", "custom-documents", name),
	}
	return c.AddDocument(doc2)
}

var (
	docQueue       = make(chan *Document, 1000)
	docQueueTicker = time.NewTicker(10 * time.Second)
	docQueueChan   = make(chan bool)
)

func (c *Config) docQueueFlush() {
	go func() {
		for {
			select {
			case <-docQueueTicker.C:
				select {
				case docQueueChan <- true:
				default:
				}
			}
		}
	}()
	go func() {
		for {
			select {
			case <-docQueueChan:
				docs := make([]*Document, 0)
			flush:
				for {
					select {
					case doc := <-docQueue:
						log.Printf("[queue] flushing doc to workspace: %s", doc.Location)
						docs = append(docs, doc)
					default:
						break flush
					}
				}
				if err := c.AddDocuments(docs); err != nil {
					log.Printf("[err] failed to add documents: %s", err)
					/*			for _, doc := range docs {
								select {
								case docQueue <- doc:
								default:
									panic("docQueue full, too many errors?")
								}
							}*/
				}
			}
		}
	}()
}

var startQueueOnce = &sync.Once{}

func (c *Config) AddDocument(doc *Document) error {
	startQueueOnce.Do(c.docQueueFlush)

	if doc.Location == "" {
		return fmt.Errorf("document location is required")
	}

	name := doc.Location

	if !strings.Contains(name, doc.ID) {
		name = strings.ReplaceAll(name, ".json", "")
		name = name + "-" + doc.ID + ".json"
	}

	doc.Location = name

	select {
	case docQueue <- doc:
	default:
		docQueueChan <- true
		time.Sleep(1 * time.Second)
		docQueue <- doc
	}
	return nil
}

func (c *Config) AddDocuments(doc []*Document) error {
	docStrings := make([]string, 0, len(doc))
	for _, d := range doc {
		if strings.TrimSpace(d.Location) != "" {
			docStrings = append(docStrings, d.Location)
		}
	}
	ue := &UpdateEmbeddings{
		Adds: docStrings,
	}
	dat, err := json.Marshal(ue)
	if err != nil {
		return err
	}
	/*
		if c.Workspace == "" {
			c.Workspace = "cia-reading-room"
		}*/

	endpoint := "v1/workspace/" + c.Workspace + "/update-embeddings"
	// log.Printf("adding document via '%s': %s", c.Endpoint+endpoint, string(dat))
	res, err := c.post(endpoint, bytes.NewBuffer(dat))
	if err == nil && res.StatusCode != 200 {
		err = fmt.Errorf("error adding document, bad status code: %s", res.Status)
	}
	return err
}
