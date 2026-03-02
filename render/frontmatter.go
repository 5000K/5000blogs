package render

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type Frontmatter struct {
	Title       string   `yaml:"title"`
	Date        string   `yaml:"date"`
	Author      string   `yaml:"author"`
	Tags        []string `yaml:"tags"`
	Description string   `yaml:"description"`
	Draft       bool     `yaml:"draft"`
	// Add more fields as needed
	Raw map[string]interface{} `yaml:",inline"`
}

// ExtractFrontmatter separates frontmatter from content
func ExtractFrontmatter(input string) (*Frontmatter, string, error) {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "---") {
		return nil, input, nil
	}

	lines := strings.Split(input, "\n")
	if len(lines) < 2 {
		return nil, input, nil
	}

	var fmLines []string
	var contentLines []string
	inFrontmatter := true
	endFound := false

	for i, line := range lines {
		if i == 0 {
			continue // skip opening ---
		}
		if inFrontmatter && strings.TrimSpace(line) == "---" {
			inFrontmatter = false
			endFound = true
			continue
		}
		if inFrontmatter {
			fmLines = append(fmLines, line)
		} else {
			contentLines = append(contentLines, line)
		}
	}

	if !endFound {
		return nil, input, fmt.Errorf("frontmatter not properly closed")
	}

	fmText := strings.Join(fmLines, "\n")
	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(fmText), &fm); err != nil {
		return nil, "", fmt.Errorf("invalid frontmatter YAML: %w", err)
	}

	return &fm, strings.Join(contentLines, "\n"), nil
}
