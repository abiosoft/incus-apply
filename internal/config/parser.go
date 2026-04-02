package config

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const DefaultFetchTimeout = 30 * time.Second

// Parser handles parsing of .incus.yaml and .incus.json configuration files.
type Parser struct {
	httpClient *http.Client
}

// NewParser creates a new config parser instance.
func NewParser(timeout time.Duration) *Parser {
	client := &http.Client{}
	if timeout > 0 {
		client.Timeout = timeout
	}
	return &Parser{httpClient: client}
}

// FileResult holds everything parsed from a single source (file, stdin, URL).
type FileResult struct {
	SourceFile string
	Vars       []*Vars
	Resources  []*Resource
}

// ParseStdin parses configuration from stdin.
func (p Parser) ParseStdin(r io.Reader) (*FileResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}

	result, err := p.parseYAML(data)
	if err != nil {
		return nil, fmt.Errorf("parsing stdin: %w", err)
	}
	result.setSourceFile("stdin")

	return result, nil
}

// ParseURL fetches and parses configuration from a URL.
func (p Parser) ParseURL(rawURL string) (*FileResult, error) {
	resp, err := p.httpClient.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", rawURL, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s: HTTP %d", rawURL, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", rawURL, err)
	}

	result, err := p.parseYAML(data)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", rawURL, err)
	}
	result.setSourceFile(rawURL)

	return result, nil
}

// ParseFile parses a single configuration file (YAML or JSON).
func (p Parser) ParseFile(path string) (*FileResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", path, err)
	}

	result, err := p.parseYAML(data)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	result.setSourceFile(path)

	return result, nil
}

// parseYAML parses YAML content, supporting multiple documents separated by '---'.
// Separates type:vars documents from resource documents.
// No interpolation is done here — the caller handles that.
func (p Parser) parseYAML(data []byte) (*FileResult, error) {
	result := &FileResult{}
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))

	for {
		// Decode into a generic map first to inspect the type
		var raw map[string]any
		err := decoder.Decode(&raw)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Skip empty documents
		if len(raw) == 0 {
			continue
		}

		typ, _ := raw["type"].(string)
		if typ == "vars" {
			var vc Vars
			if err := remarshal(raw, &vc); err != nil {
				return nil, fmt.Errorf("parsing vars document: %w", err)
			}
			result.Vars = append(result.Vars, &vc)
			continue
		}

		var res Resource
		if err := remarshal(raw, &res); err != nil {
			return nil, err
		}

		// Skip truly empty documents
		if res.Type == "" && res.Name == "" {
			continue
		}

		if err := res.Validate(); err != nil {
			return nil, err
		}
		result.Resources = append(result.Resources, &res)
	}

	return result, nil
}

func (r *FileResult) setSourceFile(source string) {
	r.SourceFile = source
	for _, v := range r.Vars {
		v.SourceFile = source
	}
	for _, c := range r.Resources {
		c.SourceFile = source
	}
}

// remarshal re-encodes a map to YAML and decodes into the target struct.
func remarshal(raw map[string]any, target any) error {
	data, err := yaml.Marshal(raw)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, target)
}
