package memory

import (
	"sync"

	"github.com/cloudwego/eino/schema"
)

const defaultMaxLength = 50

type InMemoryStore struct {
	mu        sync.RWMutex
	mem       map[int64][]schema.Message
	maxLength int
}

// create a memory store
func NewImMemoryStore() *InMemoryStore {
	store := &InMemoryStore{
		mem:       map[int64][]schema.Message{},
		maxLength: defaultMaxLength,
	}
	// Implement DB data retrieval here
	//
	return store
}

func (m *InMemoryStore) Append(userID int64, msg *schema.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// we don't store system message.
	// because system message were inserted into the memory
	if msg == nil || msg.Role == schema.System {
		return
	}
	history := append(m.mem[userID], *msg)
	if len(history) > m.maxLength {
		history = history[len(history)-m.maxLength:]
	}
	m.mem[userID] = history
}

func (m *InMemoryStore) Get(userID int64) []*schema.Message {
	m.mu.RLock()
	defer m.mu.RLock()

	history := m.mem[userID]
	output := make([]*schema.Message, 0, len(history))
	for _, msg := range history {
		output = append(output, &msg)
	}
	return output
}
