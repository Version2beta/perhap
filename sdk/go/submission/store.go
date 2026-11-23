package submission

import "context"

// Owner owns the separate private gateway admission store and trusted evaluator.
type Owner struct{}

// Client exposes only scoped Submit/Resolve, not trusted admission/evaluation writes.
type Client struct{}

func Open(ctx context.Context, path string, h Host, l Limits) (*Owner, error) { return nil, nil }
func (o *Owner) Client(scope string) *Client                                  { return nil }
func (o *Owner) Close() error                                                 { return nil }
func (c *Client) Submit(ctx context.Context, in Input, r Recipient) error     { return nil }
func (c *Client) Resolve(ctx context.Context, id string, r Recipient) error   { return nil }
func (o *Owner) Evaluate(ctx context.Context, id, attempt string) error       { return nil }
