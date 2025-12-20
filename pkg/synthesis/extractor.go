package synthesis

import (
	"encoding/json"
	"regexp"
	"strings"
)

// EntityType represents the type of extracted entity
type EntityType string

const (
	EntityTypeJIRA    EntityType = "jira"
	EntityTypeHashtag EntityType = "hashtag"
	EntityTypeMention EntityType = "mention"
	EntityTypeURL     EntityType = "url"
)

// Entity represents an extracted entity with its type and count
type Entity struct {
	Type  EntityType `json:"type"`
	Value string     `json:"value"`
	Count int        `json:"count"`
}

// EntityExtractor handles extraction and deduplication of entities from text
type EntityExtractor struct {
	jiraRegex    *regexp.Regexp
	hashtagRegex *regexp.Regexp
	mentionRegex *regexp.Regexp
	urlRegex     *regexp.Regexp
}

// NewEntityExtractor creates a new entity extractor with compiled regex patterns
func NewEntityExtractor() *EntityExtractor {
	return &EntityExtractor{
		// JIRA ticket pattern: 2-10 uppercase letters followed by dash and digits
		jiraRegex: regexp.MustCompile(`\b[A-Z]{2,10}-\d+\b`),
		
		// Hashtag pattern: # followed by alphanumeric and underscore
		hashtagRegex: regexp.MustCompile(`#[a-zA-Z0-9_]+`),
		
		// Mention pattern: @ followed by alphanumeric and underscore
		mentionRegex: regexp.MustCompile(`@[a-zA-Z0-9_]+`),
		
		// URL pattern: standard HTTP/HTTPS URLs
		urlRegex: regexp.MustCompile(`https?://[^\s<>"{}|\\^` + "`" + `\[\]]+`),
	}
}

// ExtractEntities extracts all entities from the given text and returns deduplicated results
func (e *EntityExtractor) ExtractEntities(text string) []Entity {
	entityCounts := make(map[string]int)
	entityTypes := make(map[string]EntityType)
	
	// Extract JIRA tickets
	jiraMatches := e.jiraRegex.FindAllString(text, -1)
	for _, match := range jiraMatches {
		key := strings.ToUpper(match) // Normalize case
		entityCounts[key]++
		entityTypes[key] = EntityTypeJIRA
	}
	
	// Extract hashtags
	hashtagMatches := e.hashtagRegex.FindAllString(text, -1)
	for _, match := range hashtagMatches {
		key := strings.ToLower(match) // Normalize case
		entityCounts[key]++
		entityTypes[key] = EntityTypeHashtag
	}
	
	// Extract mentions
	mentionMatches := e.mentionRegex.FindAllString(text, -1)
	for _, match := range mentionMatches {
		key := strings.ToLower(match) // Normalize case
		entityCounts[key]++
		entityTypes[key] = EntityTypeMention
	}
	
	// Extract URLs
	urlMatches := e.urlRegex.FindAllString(text, -1)
	for _, match := range urlMatches {
		key := strings.ToLower(match) // Normalize case
		entityCounts[key]++
		entityTypes[key] = EntityTypeURL
	}
	
	// Convert to deduplicated entity list
	var entities []Entity
	for value, count := range entityCounts {
		entities = append(entities, Entity{
			Type:  entityTypes[value],
			Value: value,
			Count: count,
		})
	}
	
	return entities
}

// EntitiesToJSON converts entities to JSON string for database storage
func (e *EntityExtractor) EntitiesToJSON(entities []Entity) (string, error) {
	if len(entities) == 0 {
		return "[]", nil
	}
	
	jsonBytes, err := json.Marshal(entities)
	if err != nil {
		return "[]", err
	}
	
	return string(jsonBytes), nil
}

// EntitiesFromJSON parses entities from JSON string
func (e *EntityExtractor) EntitiesFromJSON(jsonStr string) ([]Entity, error) {
	if jsonStr == "" || jsonStr == "[]" {
		return []Entity{}, nil
	}
	
	var entities []Entity
	err := json.Unmarshal([]byte(jsonStr), &entities)
	if err != nil {
		return []Entity{}, err
	}
	
	return entities, nil
}