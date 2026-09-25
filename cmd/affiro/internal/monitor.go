package internal

import (
	"sync"

	"github.com/sandgardenhq/affiro/asig"
	"github.com/sandgardenhq/affiro/asig/keylog"
)

// GlobalMonitor holds whichever keylog.Monitor Start() most recently produced, so
// that the event-polling goroutine and the shutdown paths (SIGINT/SIGTERM, GUI
// window-close) can reach it regardless of which goroutine called Start.
var GlobalMonitor MonitorHolder

type MonitorHolder struct {
	mon keylog.Monitor
	mu  sync.Mutex
}

func (h *MonitorHolder) Set(m keylog.Monitor) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.mon = m
}

func (h *MonitorHolder) Pop() (asig.Event, bool) {
	h.mu.Lock()
	m := h.mon
	h.mu.Unlock()
	if m == nil {
		return nil, false
	}
	return m.Pop()
}

func (h *MonitorHolder) Stop() {
	h.mu.Lock()
	m := h.mon
	h.mu.Unlock()
	if m != nil {
		m.Stop()
	}
}
