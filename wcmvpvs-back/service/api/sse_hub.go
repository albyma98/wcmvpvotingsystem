package api

import "sync"

// sseHub manages SSE subscriber channels keyed by an integer ID (event ID).
// Subscribers receive a signal on Broadcast and can use it to push updates to clients.
type sseHub struct {
	mu      sync.Mutex
	clients map[int]map[chan string]struct{}
}

func newSSEHub() *sseHub {
	return &sseHub{clients: make(map[int]map[chan string]struct{})}
}

// Subscribe registers a new subscriber for the given key.
// Returns a receive-only channel and an unsubscribe function that must be called when done.
func (h *sseHub) Subscribe(key int) (<-chan string, func()) {
	ch := make(chan string, 1)
	h.mu.Lock()
	if h.clients[key] == nil {
		h.clients[key] = make(map[chan string]struct{})
	}
	h.clients[key][ch] = struct{}{}
	h.mu.Unlock()

	unsub := func() {
		h.mu.Lock()
		if set, ok := h.clients[key]; ok {
			delete(set, ch)
			if len(set) == 0 {
				delete(h.clients, key)
			}
		}
		h.mu.Unlock()
	}
	return ch, unsub
}

// Broadcast sends a notification to all subscribers for the given key.
// Non-blocking: drops the signal for any subscriber whose buffer is already full.
func (h *sseHub) Broadcast(key int) {
	h.BroadcastEvent(key, "update")
}

// BroadcastEvent invia anche il tipo di aggiornamento. I consumer che non ne
// hanno bisogno continuano semplicemente a trattarlo come un normale segnale.
func (h *sseHub) BroadcastEvent(key int, eventType string) {
	h.mu.Lock()
	chs := make([]chan string, 0, len(h.clients[key]))
	for ch := range h.clients[key] {
		chs = append(chs, ch)
	}
	h.mu.Unlock()

	for _, ch := range chs {
		select {
		case ch <- eventType:
		default:
			// Un aggiornamento classifica non deve essere perso dietro a un
			// normale punto già in coda: sostituisce il segnale meno importante.
			if eventType == "standings" {
				select {
				case <-ch:
				default:
				}
				select {
				case ch <- eventType:
				default:
				}
			}
		}
	}
}
