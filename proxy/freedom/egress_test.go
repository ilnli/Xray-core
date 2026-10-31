package freedom

import (
	"context"
	"io"
	stdnet "net"
	"strings"
	"testing"

	"github.com/xtls/xray-core/common/buf"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/features/policy"
	"github.com/xtls/xray-core/transport"
	"github.com/xtls/xray-core/transport/internet/stat"
)

type egressPacketConn struct {
	net.PacketConn
	fail bool
	sent bool
}

func (c *egressPacketConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	if c.fail {
		return 0, io.ErrClosedPipe
	}
	c.sent = true
	return len(p), nil
}

func TestPacketEgressReportsSendResult(t *testing.T) {
	for _, outcome := range []string{"allowed", "blocked", "failed"} {
		t.Run(outcome, func(t *testing.T) {
			dest := net.UDPDestination(net.LocalHostIP, 5353)
			requested := net.UDPDestination(net.DomainAddress("packet.example"), 5353)
			sender := &egressPacketConn{fail: outcome == "failed"}
			var events []session.EgressEvent
			ctx := session.ContextWithEgressObserver(context.Background(), func(event session.EgressEvent) {
				if event.Outcome == "allowed" && !sender.sent {
					t.Error("allowed event before send")
				}
				events = append(events, event)
			})
			h := &Handler{}
			if outcome == "blocked" {
				h.finalRules = []*FinalRule{defaultBlockAllRule}
			}
			writer := newPacketWriter(ctx, &net.PacketConnWrapper{PacketConn: sender, Dest: dest.RawNetAddr()}, h, nil, net.Destination{}, requested, requested, nil)
			packet := buf.FromBytes([]byte("test"))
			packet.UDP = &requested
			err := writer.WriteMultiBuffer(buf.MultiBuffer{packet})
			if (err != nil) != (outcome == "failed") {
				t.Fatalf("send error: %v", err)
			}
			if len(events) != 1 || events[0].Outcome != outcome || events[0].Destination != dest || events[0].Requested.Address.String() != "packet.example" {
				t.Fatalf("unexpected events: %+v", events)
			}
			if sender.sent != (outcome == "allowed") {
				t.Fatalf("unexpected send: %v", sender.sent)
			}
		})
	}
}

type egressConn struct {
	net.Conn
	remote net.Addr
}

func (c egressConn) RemoteAddr() net.Addr {
	if c.remote != nil {
		return c.remote
	}
	return &stdnet.TCPAddr{IP: stdnet.ParseIP("203.0.113.8"), Port: 443}
}
func (egressConn) LocalAddr() net.Addr {
	return &stdnet.TCPAddr{IP: stdnet.ParseIP("192.0.2.1"), Port: 1234}
}
func (egressConn) Close() error                { return nil }
func (egressConn) Read(p []byte) (int, error)  { return 0, io.EOF }
func (egressConn) Write(p []byte) (int, error) { return len(p), nil }

type egressDialer struct {
	fail   bool
	nested bool
	conn   stat.Connection
}

func (d egressDialer) Dial(ctx context.Context, dest net.Destination) (stat.Connection, error) {
	if d.nested {
		session.ReportEgress(ctx, session.EgressEvent{Destination: dest, Outcome: "allowed"})
	}
	if d.fail {
		return nil, io.ErrClosedPipe
	}
	if d.conn != nil {
		return d.conn, nil
	}
	return egressConn{}, nil
}
func (egressDialer) DestIpAddress() net.IP                                 { return nil }
func (egressDialer) SetOutboundGateway(context.Context, *session.Outbound) {}

func TestTCPEgressReportsFinalPeerAndBlock(t *testing.T) {
	for _, outcome := range []string{"allowed", "blocked", "failed", "upstream"} {
		t.Run(outcome, func(t *testing.T) {
			requested := net.TCPDestination(net.DomainAddress("request.example"), 443)
			h := &Handler{config: &Config{}, policyManager: policy.DefaultManager{}}
			if outcome == "blocked" {
				requested.Address = net.ParseAddress("203.0.113.9")
				h.finalRules = []*FinalRule{{action: RuleAction_Block, network: allNetworks, blockDelay: &Range{}}}
			}
			h.usesDialerProxy = outcome == "upstream"
			var events []session.EgressEvent
			ctx := session.ContextWithEgressObserver(context.Background(), func(event session.EgressEvent) { events = append(events, event) })
			ctx = session.ContextWithOutbounds(ctx, []*session.Outbound{{Target: requested}})
			err := h.Process(ctx, &transport.Link{Reader: buf.NewReader(strings.NewReader("")), Writer: buf.Discard}, egressDialer{fail: outcome == "failed", nested: outcome == "upstream"})
			if (err != nil) != (outcome == "failed") {
				t.Fatalf("process error: %v", err)
			}
			if outcome == "upstream" {
				if len(events) != 0 {
					t.Fatalf("reported upstream as final: %+v", events)
				}
				return
			}
			if len(events) != 1 || events[0].Outcome != outcome || events[0].Requested != requested {
				t.Fatalf("unexpected events: %+v", events)
			}
			wantAddress := requested.Address.String()
			if outcome == "allowed" {
				wantAddress = "203.0.113.8"
			}
			if events[0].Destination.Address.String() != wantAddress {
				t.Fatalf("wrong destination: %+v", events[0])
			}
		})
	}
}

func TestUDPEgressSurvivesTimeoutOnlyContext(t *testing.T) {
	var events []session.EgressEvent
	requested := net.UDPDestination(net.DomainAddress("udp.example"), 53)
	ctx := session.ContextWithEgressObserver(context.Background(), func(event session.EgressEvent) { events = append(events, event) })
	ctx = session.ContextWithOutbounds(ctx, []*session.Outbound{{Target: requested}})
	ctx = session.ContextWithTimeoutOnly(ctx, true)
	h := &Handler{config: &Config{}, policyManager: policy.DefaultManager{}}
	conn := egressConn{remote: &stdnet.UDPAddr{IP: stdnet.ParseIP("203.0.113.53"), Port: 53}}
	err := h.Process(ctx, &transport.Link{Reader: buf.NewReader(strings.NewReader("packet")), Writer: buf.Discard}, egressDialer{conn: conn})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Outcome != "allowed" || events[0].Destination.Address.String() != "203.0.113.53" || events[0].Requested != requested {
		t.Fatalf("unexpected events: %+v", events)
	}
}
