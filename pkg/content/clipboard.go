package content

import (
	"time"

	"github.com/atotto/clipboard"
)

type ClipboardEvent struct {
	Timestamp time.Time
	Content   string
}

type Monitor struct {
	events chan ClipboardEvent
}

func NewMonitor() *Monitor {
	return &Monitor{
		events: make(chan ClipboardEvent),
	}
}

func (m *Monitor) Start() <-chan ClipboardEvent {
	go m.monitor()
	return m.events
}

func (m *Monitor) monitor() {
	lastClipboard, _ := clipboard.ReadAll()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		content, err := clipboard.ReadAll()
		if err != nil {
			continue
		}

		if content != lastClipboard {
			lastClipboard = content
			m.events <- ClipboardEvent{
				Timestamp: time.Now(),
				Content:   content,
			}
		}
	}
}
