package synthesis

import (
	"regexp"
	"strings"

	"waddle/pkg/types"
)

// Re-export types from the canonical source for local use.
type EntityType = types.EntityType
type Entity = types.Entity

// Canonical entity type constants re-exported for backward compatibility.
const (
	EntityTypeJiraTicket = types.EntityTypeJiraTicket
	EntityTypeHashtag    = types.EntityTypeHashtag
	EntityTypeMention    = types.EntityTypeMention
	EntityTypeURL        = types.EntityTypeURL
)

// Extractor extracts entities from text using regex patterns
type Extractor struct {
	patterns map[EntityType]*regexp.Regexp
}

// NewExtractor creates a new entity extractor
func NewExtractor() *Extractor {
	return &Extractor{
		patterns: map[EntityType]*regexp.Regexp{
			EntityTypeJiraTicket: regexp.MustCompile(`[A-Z]{2,10}-\d+`),
			EntityTypeHashtag:    regexp.MustCompile(`#[a-zA-Z0-9_]+`),
			EntityTypeMention:    regexp.MustCompile(`@[a-zA-Z0-9_]+`),
			EntityTypeURL:        regexp.MustCompile(`https?://[^\s]+`),
		},
	}
}

// Extract extracts and deduplicates entities from text
func (e *Extractor) Extract(text string) []Entity {
	entityCounts := make(map[string]*Entity)

	for entityType, pattern := range e.patterns {
		matches := pattern.FindAllString(text, -1)
		for _, match := range matches {
			key := string(entityType) + ":" + strings.ToLower(match)
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