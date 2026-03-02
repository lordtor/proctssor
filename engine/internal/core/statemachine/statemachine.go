package statemachine

import (
	"fmt"
	"time"
)

// StateMachine manages token state transitions
type StateMachine struct {
	// transitions holds valid state transitions
	transitions map[TokenStatus][]TokenStatus
}

// NewStateMachine creates a new state machine
func NewStateMachine() *StateMachine {
	sm := &StateMachine{
		transitions: make(map[TokenStatus][]TokenStatus),
	}

	// Define valid transitions
	sm.transitions[TokenStatusPending] = []TokenStatus{
		TokenStatusActive,
		TokenStatusWaiting,
		TokenStatusTerminated,
	}

	sm.transitions[TokenStatusActive] = []TokenStatus{
		TokenStatusCompleted,
		TokenStatusFailed,
		TokenStatusWaiting,
		TokenStatusSuspended,
		TokenStatusTerminated,
	}

	sm.transitions[TokenStatusWaiting] = []TokenStatus{
		TokenStatusActive,
		TokenStatusTerminated,
		TokenStatusSuspended,
	}

	sm.transitions[TokenStatusCompleted] = []TokenStatus{
		// Terminal state - no transitions out
	}

	sm.transitions[TokenStatusFailed] = []TokenStatus{
		TokenStatusActive, // Can retry
		TokenStatusTerminated,
	}

	sm.transitions[TokenStatusTerminated] = []TokenStatus{
		// Terminal state - no transitions out
	}

	sm.transitions[TokenStatusSuspended] = []TokenStatus{
		TokenStatusActive,
		TokenStatusTerminated,
	}

	return sm
}

// CanTransition checks if a transition from one status to another is valid
func (sm *StateMachine) CanTransition(from, to TokenStatus) bool {
	allowedStatuses, exists := sm.transitions[from]
	if !exists {
		return false
	}

	for _, status := range allowedStatuses {
		if status == to {
			return true
		}
	}

	return false
}

// Transition performs a state transition on a token
func (sm *StateMachine) Transition(token *Token, to TokenStatus, action string) error {
	from := token.Status

	// Check if transition is valid
	if !sm.CanTransition(from, to) {
		return fmt.Errorf("invalid transition from %s to %s", from, to)
	}

	// Record history
	historyEntry := TokenHistory{
		FromStatus: from,
		ToStatus:   to,
		Timestamp:  time.Now(),
		Action:     action,
	}
	token.History = append(token.History, historyEntry)

	// Update token status
	token.Status = to
	token.UpdatedAt = time.Now()

	// Set timestamps based on new status
	now := time.Now()
	switch to {
	case TokenStatusActive:
		token.StartedAt = &now
	case TokenStatusCompleted, TokenStatusFailed, TokenStatusTerminated:
		token.CompletedAt = &now
	}

	return nil
}

// Start transitions a token from pending to active
func (sm *StateMachine) Start(token *Token) error {
	return sm.Transition(token, TokenStatusActive, "start")
}

// Complete transitions a token from active to completed
func (sm *StateMachine) Complete(token *Token) error {
	return sm.Transition(token, TokenStatusCompleted, "complete")
}

// Fail transitions a token to failed state
func (sm *StateMachine) Fail(token *Token, err error) error {
	if err != nil {
		token.Error = &TokenError{
			Message: err.Error(),
		}
	}
	return sm.Transition(token, TokenStatusFailed, "fail")
}

// Terminate transitions a token to terminated state
func (sm *StateMachine) Terminate(token *Token) error {
	return sm.Transition(token, TokenStatusTerminated, "terminate")
}

// Suspend transitions a token to suspended state
func (sm *StateMachine) Suspend(token *Token) error {
	return sm.Transition(token, TokenStatusSuspended, "suspend")
}

// Resume transitions a token from suspended to active
func (sm *StateMachine) Resume(token *Token) error {
	return sm.Transition(token, TokenStatusActive, "resume")
}

// Await transitions a token to waiting state
func (sm *StateMachine) Await(token *Token, awaitType string) error {
	token.Status = TokenStatusWaiting
	token.UpdatedAt = time.Now()
	// Record in history
	historyEntry := TokenHistory{
		FromStatus: token.Status,
		ToStatus:   TokenStatusWaiting,
		Timestamp:  time.Now(),
		Action:     "await:" + awaitType,
	}
	token.History = append(token.History, historyEntry)
	return nil
}

// Trigger transitions a token from waiting to active
func (sm *StateMachine) Trigger(token *Token) error {
	return sm.Transition(token, TokenStatusActive, "trigger")
}

// GetAvailableTransitions returns available transitions from current status
func (sm *StateMachine) GetAvailableTransitions(from TokenStatus) []TokenStatus {
	return sm.transitions[from]
}

// IsTerminal checks if the status is terminal
func (sm *StateMachine) IsTerminal(status TokenStatus) bool {
	return status == TokenStatusCompleted ||
		status == TokenStatusFailed ||
		status == TokenStatusTerminated
}

// TokenStateMachine provides functions for token state management
var TokenStateMachine = NewStateMachine()
