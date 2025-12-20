package synthesis

import (
	"regexp"
	"strings"
)

// Entity represents an extracted entity
type Entity struct {
	Value string     `json:"value"`
	Type  EntityType `json:"type"`
	Count int        `json:"count"`
}

// EntityType represents the type of entity
type EntityType string

const (
	EntityTypeJIRA    EntityType = "jira"
	EntityTypeHashtag EntityType = "hashtag"
	EntityTypeMention EntityType = "mention"
	EntityTypeURL     EntityType = "url"
)

// Extractor extracts entities from text using regex patterns
type Extractor struct {
	patterns map[EntityType]*regexp.Regexp
}

// NewExtractor creates a new entity extractor
func NewExtractor() *Extractor {
	return &Extractor{
		patterns: map[EntityType]*regexp.Regexp{
			EntityTypeJIRA:    regexp.MustCompile(`[A-Z]{2,10}-\d+`),
			EntityTypeHashtag: regexp.MustCompile(`#[a-zA-Z0-9_]+`),
			EntityTypeMention: regexp.MustCompile(`@[a-zA-Z0-9_]+`),
			EntityTypeURL:     regexp.MustCompile(`https?://[^\s]+`),
		},
	}
}

// Extract extracts and deduplicates entities from text
func (e *Extractor) Extract(text string) []Entity {
	entityCounts := make(map[string]*Entity)

	for entityType, pattern := range e.patterns {
		matches := pattern.FindAllString(text, -1)
		for _, match := range matches {
			key := strings.ToLower(match)
			if entity, exists := entityCounts[key]; exists {
				entity.Count++
			} else {
				entityCounts[key] = &Entity{
					Value: match,
					Type:  entityType,
					Count: 1,
				}
			}
		}
	}

	// Convert map to slice
	var entities []Entity
	for _, entity := range entityCounts {
		entities = append(entities, *entity)
	}

	return entities
}