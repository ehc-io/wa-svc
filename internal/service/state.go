package service

import (
	"log"
	"sync"
)

// State represents the connection state of the WhatsApp service.
type State string

const (
	StateUnauthenticated State = "unauthenticated"
	StatePairing         State = "pairing"
	StateConnecting      State = "connecting"
	StateConnected       State = "connected"
	StateDisconnected    State = "disconnected"
	StateError           State = "error"
)

// String returns the string representation of the state.
func (s State) String() string {
	return string(s)
}

// IsReady returns true if the service is ready to send/receive messages.
func (s State) IsReady() bool {
	return s == StateConnected
}

// StateMachine manages the connection state with thread-safe transitions.
type StateMachine struct {
	mu        sync.RWMutex
	state     State
	lastError error
	qrCode    string
	listeners []func(old, new State)
}

// NewStateMachine creates a new state machine starting in unauthenticated state.
func NewStateMachine() *StateMachine {
	return &StateMachine{
		state: StateUnauthenticated,
	}
}

// State returns the current state.
func (sm *StateMachine) State() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state
}

// SetState transitions to a new state.
func (sm *StateMachine) SetState(newState State) {
	sm.mu.Lock()
	oldState := sm.state
	sm.state = newState
	if newState != StatePairing {
		sm.qrCode = ""
	}
	if newState != StateError {
		sm.lastError = nil
	}
	listeners := sm.listeners
	sm.mu.Unlock()

	// Notify listeners outside the lock
	if oldState != newState {
		for _, fn := range listeners {
			fn(oldState, newState)
		}
	}
}

// SetError sets an error state with the given error.
func (sm *StateMachine) SetError(err error) {
	sm.mu.Lock()
	oldState := sm.state
	sm.state = StateError
	sm.lastError = err
	sm.qrCode = ""
	listeners := sm.listeners
	sm.mu.Unlock()

	if oldState != StateError {
		for _, fn := range listeners {
			fn(oldState, StateError)
		}
	}
}

// LastError returns the last error if in error state.
func (sm *StateMachine) LastError() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.lastError
}

// SetQRCode stores the current QR code for pairing.
func (sm *StateMachine) SetQRCode(code string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.qrCode = code
	sm.state = StatePairing
	log.Printf("[State] QR code set (length: %d), state -> pairing", len(code))
}

// QRCode returns the current QR code if in pairing state.
func (sm *StateMachine) QRCode() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.state == StatePairing {
		return sm.qrCode
	}
	return ""
}

// ClearQRCode clears the stored QR code.
func (sm *StateMachine) ClearQRCode() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.qrCode = ""
}

// OnStateChange registers a callback for state changes.
func (sm *StateMachine) OnStateChange(fn func(old, new State)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.listeners = append(sm.listeners, fn)
}

// StatusInfo returns a snapshot of the current status.
func (sm *StateMachine) StatusInfo() StatusInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	info := StatusInfo{
		State:   sm.state,
		Ready:   sm.state.IsReady(),
		HasQR:   sm.qrCode != "",
	}
	if sm.lastError != nil {
		info.Error = sm.lastError.Error()
	}
	return info
}

// StatusInfo holds status information for API responses.
type StatusInfo struct {
	State State  `json:"state"`
	Ready bool   `json:"ready"`
	HasQR bool   `json:"has_qr"`
	Error string `json:"error,omitempty"`
}
