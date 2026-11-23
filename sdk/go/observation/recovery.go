package observation

import "context"

// Recovery owns one bounded reconnect slot. Host excludes previous live owner.
type Recovery struct{}
type Recovered struct{}

// NewRecovery reconciles only its exclusively owned namespace's stale plans.
func NewRecovery(ctx context.Context, source RecoverySource, gate *Gate, proofs ProofAuthority, owner string, l RecoveryLimits) (*Recovery, error) {
	return nil, nil
}
func (m *Recovery) Open(ctx context.Context, b Binding, r RecoveryRecipient) (*Recovered, error) {
	return nil, nil
}
func (m *Recovery) Resume(ctx context.Context, b Binding, token string, r RecoveryRecipient) (*Recovered, error) {
	return nil, nil
}
func (m *Recovery) Replace(ctx context.Context, b Binding, r RecoveryRecipient) (*Recovered, error) {
	return nil, nil
}
func (m *Recovery) Close() error  { return nil }
func (s *Recovered) Step() error  { return nil }
func (s *Recovered) Close() error { return nil }
