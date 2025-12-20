package synthesis

import (
	"reflect"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestEntityExtractor_BasicExtraction(t *testing.T) {
	extractor := NewEntityExtractor()
	
	testCases := []struct {
		name     string
		text     string
		expected []Entity
	}{
		{
			name: "JIRA tickets",
			text: "Working on PROJ-123 and TEAM-456",
			expected: []Entity{
				{Type: EntityTypeJIRA, Value: "PROJ-123", Count: 1},
				{Type: EntityTypeJIRA, Value: "TEAM-456", Count: 1},
			},
		},
		{
			name: "Hashtags",
			text: "This is #awesome and #cool #awesome",
			expected: []Entity{
				{Type: EntityTypeHashtag, Value: "#awesome", Count: 2},
				{Type: EntityTypeHashtag, Value: "#cool", Count: 1},
			},
		},
		{
			name: "Mentions",
			text: "Thanks @john and @jane @john",
			expected: []Entity{
				{Type: EntityTypeMention, Value: "@john", Count: 2},
				{Type: EntityTypeMention, Value: "@jane", Count: 1},
			},
		},
		{
			name: "URLs",
			text: "Check https://example.com and https://test.org",
			expected: []Entity{
				{Type: EntityTypeURL, Value: "https://example.com", Count: 1},
				{Type: EntityTypeURL, Value: "https://test.org", Count: 1},
			},
		},
		{
			name: "Mixed entities",
			text: "PROJ-123 #bug @dev https://example.com/tickets",
			expected: []Entity{
				{Type: EntityTypeJIRA, Value: "PROJ-123", Count: 1},
				{Type: EntityTypeHashtag, Value: "#bug", Count: 1},
				{Type: EntityTypeMention, Value: "@dev", Count: 1},
				{Type: EntityTypeURL, Value: "https://example.com/tickets", Count: 1},
			},
		},
		{
			name: "Empty text",
			text: "",
			expected: []Entity{},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entities := extractor.ExtractEntities(tc.text)
			
			if len(entities) != len(tc.expected) {
				t.Errorf("Expected %d entities, got %d", len(tc.expected), len(entities))
				return
			}
			
			// Convert to maps for easier comparison
			actualMap := make(map[string]Entity)
			for _, e := range entities {
				actualMap[e.Value] = e
			}
			
			for _, expected := range tc.expected {
				actual, exists := actualMap[expected.Value]
				if !exists {
					t.Errorf("Expected entity %s not found", expected.Value)
					continue
				}
				
				if actual.Type != expected.Type {
					t.Errorf("Entity %s: expected type %s, got %s", expected.Value, expected.Type, actual.Type)
				}
				
				if actual.Count != expected.Count {
					t.Errorf("Entity %s: expected count %d, got %d", expected.Value, expected.Count, actual.Count)
				}
			}
		})
	}
}

func TestEntityExtractor_JSONRoundTrip(t *testing.T) {
	extractor := NewEntityExtractor()
	
	original := []Entity{
		{Type: EntityTypeJIRA, Value: "PROJ-123", Count: 2},
		{Type: EntityTypeHashtag, Value: "#test", Count: 1},
		{Type: EntityTypeMention, Value: "@user", Count: 3},
		{Type: EntityTypeURL, Value: "https://example.com", Count: 1},
	}
	
	// Convert to JSON
	jsonStr, err := extractor.EntitiesToJSON(original)
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}
	
	// Convert back from JSON
	parsed, err := extractor.EntitiesFromJSON(jsonStr)
	if err != nil {
		t.Fatalf("Failed to parse from JSON: %v", err)
	}
	
	// Verify round-trip
	if len(parsed) != len(original) {
		t.Errorf("Expected %d entities after round-trip, got %d", len(original), len(parsed))
		return
	}
	
	parsedMap := make(map[string]Entity)
	for _, e := range parsed {
		parsedMap[e.Value] = e
	}
	
	for _, expected := range original {
		actual, exists := parsedMap[expected.Value]
		if !exists {
			t.Errorf("Entity %s missing after round-trip", expected.Value)
			continue
		}
		
		if actual.Type != expected.Type || actual.Count != expected.Count {
			t.Errorf("Entity %s changed after round-trip: expected %+v, got %+v", expected.Value, expected, actual)
		}
	}
}

// Property 12: Entity Extraction with Deduplication
func TestProperty_EntityExtractionWithDeduplication(t *testing.T) {
	extractor := NewEntityExtractor()
	
	properties := gopter.NewProperties(nil)
	
	properties.Property("Entity extraction deduplicates correctly", prop.ForAll(
		func(jiraTickets []string, hashtags []string, mentions []string, urls []string) bool {
			// Build test text with duplicates
			var textParts []string
			
			// Add JIRA tickets (ensure valid format)
			for _, ticket := range jiraTickets {
				if len(ticket) >= 3 {
					validTicket := strings.ToUpper(ticket[:2]) + "-123"
					textParts = append(textParts, validTicket, validTicket) // Add twice for deduplication test
				}
			}
			
			// Add hashtags
			for _, tag := range hashtags {
				if len(tag) > 0 {
					validTag := "#" + strings.ReplaceAll(tag, " ", "_")
					textParts = append(textParts, validTag, validTag) // Add twice
				}
			}
			
			// Add mentions
			for _, mention := range mentions {
				if len(mention) > 0 {
					validMention := "@" + strings.ReplaceAll(mention, " ", "_")
					textParts = append(textParts, validMention, validMention) // Add twice
				}
			}
			
			// Add URLs
			for _, url := range urls {
				if len(url) > 0 {
					validURL := "https://" + strings.ReplaceAll(url, " ", "") + ".com"
					textParts = append(textParts, validURL, validURL) // Add twice
				}
			}
			
			text := strings.Join(textParts, " ")
			entities := extractor.ExtractEntities(text)
			
			// Verify deduplication: each entity should have count >= 1
			for _, entity := range entities {
				if entity.Count < 1 {
					return false
				}
				
				// For our test case where we add each entity twice, count should be 2
				if len(textParts) > 0 && entity.Count != 2 {
					return false
				}
			}
			
			// Verify no duplicate entities in result
			seen := make(map[string]bool)
			for _, entity := range entities {
				if seen[entity.Value] {
					return false // Duplicate found
				}
				seen[entity.Value] = true
			}
			
			return true
		},
		gen.SliceOfN(3, gen.AlphaString()),     // jiraTickets
		gen.SliceOfN(3, gen.AlphaString()),     // hashtags
		gen.SliceOfN(3, gen.AlphaString()),     // mentions
		gen.SliceOfN(3, gen.AlphaString()),     // urls
	))
	
	properties.TestingRun(t)
}

// Property 13: Entity JSON Storage Round-Trip
func TestProperty_EntityJSONStorageRoundTrip(t *testing.T) {
	extractor := NewEntityExtractor()
	
	properties := gopter.NewProperties(nil)
	
	properties.Property("Entity JSON round-trip preserves data", prop.ForAll(
		func(entityData []struct {
			Type  int
			Value string
			Count int
		}) bool {
			// Convert test data to entities
			var original []Entity
			for _, data := range entityData {
				if data.Count <= 0 || len(data.Value) == 0 {
					continue // Skip invalid data
				}
				
				var entityType EntityType
				switch data.Type % 4 {
				case 0:
					entityType = EntityTypeJIRA
				case 1:
					entityType = EntityTypeHashtag
				case 2:
					entityType = EntityTypeMention
				case 3:
					entityType = EntityTypeURL
				}
				
				original = append(original, Entity{
					Type:  entityType,
					Value: data.Value,
					Count: data.Count,
				})
			}
			
			// Convert to JSON and back
			jsonStr, err := extractor.EntitiesToJSON(original)
			if err != nil {
				return false
			}
			
			parsed, err := extractor.EntitiesFromJSON(jsonStr)
			if err != nil {
				return false
			}
			
			// Verify round-trip preservation
			if len(parsed) != len(original) {
				return false
			}
			
			// Create maps for comparison
			originalMap := make(map[string]Entity)
			for _, e := range original {
				originalMap[e.Value] = e
			}
			
			parsedMap := make(map[string]Entity)
			for _, e := range parsed {
				parsedMap[e.Value] = e
			}
			
			// Compare all entities
			for value, originalEntity := range originalMap {
				parsedEntity, exists := parsedMap[value]
				if !exists {
					return false
				}
				
				if originalEntity.Type != parsedEntity.Type || originalEntity.Count != parsedEntity.Count {
					return false
				}
			}
			
			return true
		},
		gen.SliceOfN(10, gen.Struct(reflect.TypeOf(struct {
			Type  int
			Value string
			Count int
		}{}), map[string]gopter.Gen{
			"Type":  gen.IntRange(0, 10),
			"Value": gen.AlphaString(),
			"Count": gen.IntRange(1, 100),
		})),
	))
	
	properties.TestingRun(t)
}