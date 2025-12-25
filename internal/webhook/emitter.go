package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Event represents a webhook event payload.
type Event struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// Config holds webhook configuration.
type Config struct {
	URL        string
	Secret     string
	MaxRetries int
	Timeout    time.Duration
}

// Emitter handles webhook delivery with retry logic.
type Emitter struct {
	config     Config
	client     *http.Client
	queue      chan *queuedEvent
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	maxWorkers int
}

type queuedEvent struct {
	event   *Event
	retries int
}

// NewEmitter creates a new webhook emitter.
func NewEmitter(cfg Config) *Emitter {
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	e := &Emitter{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		queue:      make(chan *queuedEvent, 1000),
		ctx:        ctx,
		cancel:     cancel,
		maxWorkers: 4,
	}

	return e
}

// Start begins processing the webhook queue.
func (e *Emitter) Start() {
	for i := 0; i < e.maxWorkers; i++ {
		e.wg.Add(1)
		go e.worker()
	}
	log.Printf("[Webhook] Started %d workers", e.maxWorkers)
}

// Stop gracefully shuts down the emitter.
func (e *Emitter) Stop() {
	e.cancel()
	close(e.queue)
	e.wg.Wait()
	log.Println("[Webhook] Stopped")
}

// Emit queues an event for delivery.
func (e *Emitter) Emit(eventType string, data interface{}) {
	if e.config.URL == "" {
		return
	}

	event := &Event{
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Data:      data,
	}

	select {
	case e.queue <- &queuedEvent{event: event, retries: 0}:
	default:
		log.Println("[Webhook] Queue full, dropping event")
	}
}

// worker processes events from the queue.
func (e *Emitter) worker() {
	defer e.wg.Done()

	for {
		select {
		case <-e.ctx.Done():
			return
		case qe, ok := <-e.queue:
			if !ok {
				return
			}
			e.deliver(qe)
		}
	}
}

// deliver attempts to send the webhook with retries.
func (e *Emitter) deliver(qe *queuedEvent) {
	payload, err := json.Marshal(qe.event)
	if err != nil {
		log.Printf("[Webhook] Failed to marshal event: %v", err)
		return
	}

	for attempt := 0; attempt <= qe.retries+e.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s, ...
			backoff := time.Duration(1<<(attempt-1)) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			select {
			case <-e.ctx.Done():
				return
			case <-time.After(backoff):
			}
		}

		err := e.send(payload)
		if err == nil {
			if attempt > 0 {
				log.Printf("[Webhook] Event %s delivered after %d retries", qe.event.Type, attempt)
			}
			return
		}

		log.Printf("[Webhook] Delivery attempt %d failed: %v", attempt+1, err)
	}

	log.Printf("[Webhook] Event %s dropped after %d attempts", qe.event.Type, e.config.MaxRetries+1)
}

// send performs the actual HTTP request.
func (e *Emitter) send(payload []byte) error {
	req, err := http.NewRequestWithContext(e.ctx, http.MethodPost, e.config.URL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "wasvc-webhook/1.0")

	// Add HMAC signature if secret is configured
	if e.config.Secret != "" {
		signature := computeHMAC(payload, e.config.Secret)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

// computeHMAC generates an HMAC-SHA256 signature.
func computeHMAC(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// IsConfigured returns true if a webhook URL is set.
func (e *Emitter) IsConfigured() bool {
	return e.config.URL != ""
}
