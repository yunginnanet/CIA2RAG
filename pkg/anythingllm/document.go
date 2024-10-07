package anythingllm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type UploadLink struct {
	Link string `json:"link"`
}

type Document struct {
	ID                 string `json:"id"`
	URL                string `json:"url"`
	Title              string `json:"title"`
	DocAuthor          string `json:"docAuthor"`
	Description        string `json:"description"`
	DocSource          string `json:"docSource"`
	ChunkSource        string `json:"chunkSource"`
	Published          string `json:"published"`
	WordCount          int    `json:"wordCount"`
	PageContent        string `json:"pageContent"`
	TokenCountEstimate int    `json:"token_count_estimate"`
	Location           string `json:"location"`
}

type UploadLinkResponse struct {
	Success   bool        `json:"success"`
	Error     interface{} `json:"error"`
	Documents []Document  `json:"documents"`
}

var ErrAccessDenied = errors.New("access denied")

type RemoveDocument struct {
	Names []string `json:"names"`
}

func (c *Config) DeleteDocument(location string) error {
	rd := &RemoveDocument{Names: []string{location}}
	dat, _ := json.Marshal(rd)
	res, err := c.delete("v1/system/remove-documents", bytes.NewReader(dat))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		_ = res.Body.Close()
		return fmt.Errorf("failed to remove document: %s", http.StatusText(res.StatusCode))
	}
	return nil
}

var ErrDuplicate = errors.New("already seen link")

func (c *Config) UploadLink(s string) (*Document, error) {

	if c.hasSeenURL(s) {
		return nil, ErrDuplicate
	}

	c.markSeenURL(s)

	l := &UploadLink{Link: s}
	dat, _ := json.Marshal(l)
	strings.NewReader(s)
	res, err := c.post("v1/document/upload-link", bytes.NewReader(dat))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		_ = res.Body.Close()
		return nil, fmt.Errorf("failed to upload link: %s", http.StatusText(res.StatusCode))
	}
	up := &UploadLinkResponse{}
	data, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if err = json.Unmarshal(data, up); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if len(up.Documents) == 0 {
		return nil, errors.New("no documents uploaded")
	}

	if strings.HasPrefix(up.Documents[0].PageContent, "Access Denied") {
		return &up.Documents[0], ErrAccessDenied
	}

	return &up.Documents[0], nil
}

// link://https://[...]

type Item struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	ID          string `json:"id"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	ChunkSource string `json:"chunkSource"`
	WordCount   int    `json:"wordCount"`
	Cached      bool   `json:"cached"`
	Items       []Item `json:"items"`
}

type DocumentsResponse struct {
	LocalFiles struct {
		Name  string `json:"name"`
		Type  string `json:"type"`
		Items []Item `json:"items"`
	} `json:"localFiles"`
}

type DocumentsFolder map[string][]Item
type Seen map[string]bool

func DocsToFolders(resp *DocumentsResponse) DocumentsFolder {
	folders := make(DocumentsFolder)
	count := 0
	for _, item := range resp.LocalFiles.Items {
		folders[item.Name] = item.Items
		count++
		for _, subItem := range item.Items {
			if subItem.Type == "folder" {
				folders[item.Name] = append(folders[item.Name], subItem.Items...)
				count += len(subItem.Items)
			} else {
				folders[item.Name] = append(folders[item.Name], subItem)
				count++
			}
			for _, subSubItem := range subItem.Items {
				if subSubItem.Type == "folder" {
					folders[item.Name] = append(folders[item.Name], subSubItem.Items...)
					count += len(subSubItem.Items)
				} else {
					folders[item.Name] = append(folders[item.Name], subSubItem)
					count++
				}
			}
		}
	}

	log.Printf("total existing documents observed for dedupe purposes: %d", count)

	return folders
}

func (c *Config) GetDocuments() (DocumentsFolder, error) {
	log.Println("getting documents")
	res, err := c.get("v1/documents")
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		_ = res.Body.Close()
		return nil, fmt.Errorf("failed to get documents: %s", http.StatusText(res.StatusCode))
	}
	docs := &DocumentsResponse{}
	data, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if err = json.Unmarshal(data, docs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	docFolders := DocsToFolders(docs)

	return docFolders, nil
}
