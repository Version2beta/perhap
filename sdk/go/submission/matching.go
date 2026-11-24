package submission

import "context"

// ResolveMatching checks normalized retained immutable meaning without creating
// receipts/admission/evaluation. Mismatch is a fenced identity-conflict reply.
// It is read-only: it never calls Admission and never writes records.
func (c *Client) ResolveMatching(ctx context.Context, in Input, r Recipient) error {
	o := c.owner
	done, err := o.enter()
	if err != nil {
		return err
	}
	defer done()
	ctx, stop := o.bound(ctx)
	defer stop()
	if r == nil {
		return ErrInvalid
	}
	if in.Event.Context != c.scope {
		return ErrDenied
	}
	if err := validateEvent(in.Event, in.Schema); err != nil {
		return err
	}
	actor, err := o.host.Authorize(ctx, "match", in.Event.ID, in.Event.Context, in.AttributionBinding)
	if err != nil {
		return publicErr(err, ErrDenied)
	}
	codec, err := o.host.Codec(in.Schema)
	if err != nil {
		return publicErr(err, ErrInvalid)
	}
	norm, err := codec.Normalize(cloneEvent(in.Event))
	if err != nil || !sameIdentity(in.Event, norm) {
		return ErrInvalid
	}
	ev, err := newStoredEvent(norm, in.Schema, actor.Asserting)
	if err != nil {
		return err
	}
	// An envelope over MaxEventBytes could never have been retained, so it cannot match.
	if n, err := envelopeBytes(ev); err != nil {
		return err
	} else if n > o.limits.MaxEventBytes {
		return ErrCapacity
	}
	reply, err := o.match(c.scope, ev)
	if err != nil {
		return err
	}
	return o.release(ctx, actor, in.Event.ID, reply, r)
}

// match compares the normalized request with the retained semantic identity of the same
// event under mu. Absent or other-scope knowledge is no-retained-knowledge; a mismatch
// in any semantic field, including schema and attribution, is identity-conflict with no
// facts. The reply is released by the caller after mu is dropped.
func (o *Owner) match(scope string, ev storedEvent) (Reply, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if err := o.usable(); err != nil {
		return Reply{}, err
	}
	cur, found := o.st.entries[ev.ID]
	if !found || cur.ev.Context != scope {
		return Reply{Kind: "no-retained-knowledge"}, nil
	}
	if !sameSemantic(cur.ev, ev) {
		return Reply{Kind: "identity-conflict"}, nil
	}
	return Reply{Kind: "facts", Facts: cur.facts}, nil
}
