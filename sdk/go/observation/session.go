package observation

import "context"

// Gate serializes the actual synchronous release with trusted local revocation.
type Gate struct{}

// NewGate constructs a local release boundary over injected authority.
func NewGate(authority Authority, binding Binding) *Gate { return nil }

// Revoke retires one exact authorization epoch; later release is forbidden.
func (g *Gate) Revoke() {}

// Release holds the revocation fence through the recipient callback.
func (g *Gate) Release(ctx context.Context, binding Binding, frame Frame, recipient Recipient) error {
	return nil
}

// Manager owns quotas and dependencies; each returned Session owns its durable pin.
type Manager struct{}

// NewManager creates bounded bootstrap/session orchestration, not a transport.
func NewManager(journal Journal, gate *Gate, limits Limits) (*Manager, error) { return nil, nil }

// Open authorizes before preparing any snapshot or exposing its metadata.
func (m *Manager) Open(ctx context.Context, binding Binding, recipient Recipient) (*Session, error) {
	return nil, nil
}

// Session owns one cancellation fence, serialized step and cleanup obligation.
type Session struct{}

// Step releases snapshot once, at most one page of successors, then a finite barrier.
func (s *Session) Step() error { return nil }

// Close cancels and joins in-flight synchronous work, then releases the retention pin.
// It is idempotent on success; failed cleanup remains owned/retryable.
func (s *Session) Close() error { return nil }
