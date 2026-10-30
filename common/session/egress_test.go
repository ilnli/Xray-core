package session

import (
	"context"
	"testing"

	"github.com/xtls/xray-core/common/net"
)

func TestEgressObserverIsContextScoped(t *testing.T) {
	var events []EgressEvent
	ctx := ContextWithEgressObserver(context.Background(), func(event EgressEvent) {
		events = append(events, event)
	})
	event := EgressEvent{Destination: net.TCPDestination(net.LocalHostIP, 443), Outcome: "allowed"}
	ReportEgress(context.Background(), event)
	ReportEgress(ContextWithEgressObserver(ctx, nil), event)
	ReportEgress(context.WithValue(ctx, struct{}{}, true), event)
	if len(events) != 1 || events[0] != event {
		t.Fatalf("unexpected events: %+v", events)
	}
}
