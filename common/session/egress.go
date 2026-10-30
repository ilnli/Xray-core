package session

import (
	"context"

	"github.com/xtls/xray-core/common/net"
)

// EgressEvent describes a final outbound connection or datagram without payload.
type EgressEvent struct {
	// Destination is the connected peer or datagram target. For blocked or failed
	// attempts it is the attempted target, which may remain a domain if unresolved.
	Destination net.Destination
	// Requested is the target before outbound address overrides and resolution.
	Requested net.Destination
	// Outcome is "allowed", "blocked", or "failed". Allowed TCP events follow
	// connection establishment; allowed UDP events follow a successful send.
	Outcome string
}

type egressObserverKey struct{}

// ContextWithEgressObserver installs an observer for this context and its children.
// It runs synchronously and must be fast and safe for concurrent calls. Events
// may repeat (especially for UDP); consumers own any bounded deduplication.
// A nil observer disables reporting in the derived context.
func ContextWithEgressObserver(ctx context.Context, observer func(EgressEvent)) context.Context {
	return context.WithValue(ctx, egressObserverKey{}, observer)
}

// ReportEgress reports an event if an observer is installed.
func ReportEgress(ctx context.Context, event EgressEvent) {
	if observer, ok := ctx.Value(egressObserverKey{}).(func(EgressEvent)); ok && observer != nil {
		observer(event)
	}
}
