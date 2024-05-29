package messagebus

import (
	"github.com/burenotti/go_health_backend/internal/domain"
	"log/slog"
	"sync"
)

type EventHandler func(event domain.Event) error

type MessageBus struct {
	logger   *slog.Logger
	handlers map[string][]EventHandler
	wg       sync.WaitGroup
}

func New(logger *slog.Logger) *MessageBus {
	return &MessageBus{
		logger:   logger,
		handlers: make(map[string][]EventHandler),
		wg:       sync.WaitGroup{},
	}
}

func (b *MessageBus) Register(eventType string, handler EventHandler) {
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

func (b *MessageBus) PublishEvents(events ...domain.Event) error {
	for _, event := range events {
		for _, handler := range b.handlers[event.Type()] {
			b.wg.Add(1)
			go func() {
				defer b.wg.Done()
				if err := handler(event); err != nil {
					b.logger.Error("failed to handle event", "type", event.Type(), "err", err)
				}
			}()
		}
	}
	return nil
}

func (b *MessageBus) Close() {
	b.wg.Wait()
}
