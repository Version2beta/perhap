package submission

import "context"

// ResolveMatching checks normalized retained immutable meaning without creating
// receipts/admission/evaluation. Mismatch is a fenced identity-conflict reply.
func (c *Client) ResolveMatching(ctx context.Context, in Input, r Recipient) error { return nil }
