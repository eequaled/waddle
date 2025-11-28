package processing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"ideathon/pkg/ai"
)

type ActivityBlock struct {
	ID           string    `json:"id"`
	StartTime    time.Time `json:"startTime"`
	EndTime      time.Time `json:"endTime"`
	OCRText      string    `json:"ocrText"`
	MicroSummary string    `json:"microSummary"`
}

type MemoryManager struct {
	SessionRoot string
	AI          *ai.OllamaClient
}

func NewMemoryManager(root string, aiClient *ai.OllamaClient) *MemoryManager {
	return &MemoryManager{
		SessionRoot: root,
		AI:          aiClient,
	}
}

// ProcessMemories scans all sessions and creates activity blocks
func (mm *MemoryManager) ProcessMemories() error {
	// Similar traversal to BatchProcessor
	entries, err := os.ReadDir(mm.SessionRoot)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if _, err := time.Parse("2006-01-02", entry.Name()); err == nil {
				mm.ProcessSession(filepath.Join(mm.SessionRoot, entry.Name()), entry.Name())
			}
		}
	}
	return nil
}

func (mm *MemoryManager) ProcessSession(sessionDir, dateStr string) {
	entries, _ := os.ReadDir(sessionDir)
	for _, entry := range entries {
		if entry.IsDir() {
			mm.ProcessApp(filepath.Join(sessionDir, entry.Name()), dateStr)
		}
	}
}

func (mm *MemoryManager) ProcessApp(appDir, dateStr string) {
	ocrPath := filepath.Join(appDir, "ocr.txt")
	content, err := os.ReadFile(ocrPath)
	if err != nil {
		return // No OCR data yet
	}

	// 1. Parse OCR Text
	entries := parseOCREntries(string(content), dateStr)
	if len(entries) == 0 {
		return
	}

	// 2. Group into 2-minute blocks (for testing/demo)
	blocks := groupEntriesByTime(entries, 2*time.Minute)

	// 3. Ensure "blocks" directory exists
	blocksDir := filepath.Join(appDir, "blocks")
	os.MkdirAll(blocksDir, 0755)

	// 4. Process each block
	for _, block := range blocks {
		blockID := block.StartTime.Format("15-04")
		blockFile := filepath.Join(blocksDir, blockID+".json")

		// Skip if already exists (immutable history)
		if _, err := os.Stat(blockFile); err == nil {
			continue
		}

		// Generate Summary
		summary, err := mm.AI.Summarize(filepath.Base(appDir), block.OCRText)
		if err != nil {
			fmt.Printf("AI Summary failed for %s: %v\n", blockID, err)
			summary = "Summary unavailable"
		}
		block.MicroSummary = summary
		block.ID = blockID

		// Save JSON
		data, _ := json.MarshalIndent(block, "", "  ")
		os.WriteFile(blockFile, data, 0644)
		// fmt.Printf("Created memory block: %s\n", blockFile)
	}
}

type OCREntry struct {
	Time time.Time
	Text string
}

// parseOCREntries parses [HH-MM-SS] entries from ocr.txt
func parseOCREntries(content, dateStr string) []OCREntry {
	var entries []OCREntry
	// Regex to match [HH-MM-SS]
	re := regexp.MustCompile(`\[(\d{2}-\d{2}-\d{2})\]`)

	// Split by timestamp markers
	indexes := re.FindAllStringIndex(content, -1)
	if len(indexes) == 0 {
		return entries
	}

	for i, idx := range indexes {
		timestampStr := content[idx[0]+1 : idx[1]-1] // Remove []

		// Parse full time
		fullTimeStr := fmt.Sprintf("%s %s", dateStr, timestampStr)
		t, err := time.Parse("2006-01-02 15-04-05", fullTimeStr)
		if err != nil {
			continue
		}

		// Get text until next marker or end
		startText := idx[1]
		endText := len(content)
		if i < len(indexes)-1 {
			endText = indexes[i+1][0]
		}

		text := strings.TrimSpace(content[startText:endText])
		if text != "" {
			entries = append(entries, OCREntry{Time: t, Text: text})
		}
	}
	return entries
}

// groupEntriesByTime groups entries into fixed intervals
func groupEntriesByTime(entries []OCREntry, interval time.Duration) []*ActivityBlock {
	var blocks []*ActivityBlock
	if len(entries) == 0 {
		return blocks
	}

	// Start bucket at the first entry's time, rounded down to interval?
	// Or just simple greedy grouping?
	// Let's do simple greedy grouping starting from first entry

	currentBlock := &ActivityBlock{
		StartTime: entries[0].Time,
		EndTime:   entries[0].Time,
		OCRText:   entries[0].Text,
	}

	// Define the end boundary of the first block
	// E.g. if start is 09:02, and interval is 15m, end is 09:17
	blockEnd := currentBlock.StartTime.Add(interval)

	for i := 1; i < len(entries); i++ {
		entry := entries[i]

		if entry.Time.Before(blockEnd) {
			// Add to current block
			currentBlock.OCRText += "\n" + entry.Text
			currentBlock.EndTime = entry.Time
		} else {
			// Finalize current block
			blocks = append(blocks, currentBlock)

			// Start new block
			currentBlock = &ActivityBlock{
				StartTime: entry.Time,
				EndTime:   entry.Time,
				OCRText:   entry.Text,
			}
			blockEnd = currentBlock.StartTime.Add(interval)
		}
	}

	// Append last block
	blocks = append(blocks, currentBlock)
	return blocks
}
