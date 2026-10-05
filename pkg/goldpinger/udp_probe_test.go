package goldpinger

import (
	"encoding/binary"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func startTestEchoListener(t *testing.T) (int, func()) {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := pc.LocalAddr().(*net.UDPAddr).Port

	go func() {
		buf := make([]byte, udpMaxPacketSize)
		for {
			n, addr, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			if n >= udpHeaderSize {
				magic := binary.BigEndian.Uint32(buf[0:4])
				if magic == udpMagic {
					pc.WriteTo(buf[:n], addr)
				}
			}
		}
	}()

	return port, func() { pc.Close() }
}

// startLossyEchoListener echoes back packets but drops every dropEveryN-th
// packet (1-indexed). For example, dropEveryN=3 drops packets 3, 6, 9, etc.
func startLossyEchoListener(t *testing.T, dropEveryN int) (int, func()) {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := pc.LocalAddr().(*net.UDPAddr).Port

	var counter atomic.Int64

	go func() {
		buf := make([]byte, udpMaxPacketSize)
		for {
			n, addr, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			if n >= udpHeaderSize {
				magic := binary.BigEndian.Uint32(buf[0:4])
				if magic == udpMagic {
					seq := counter.Add(1)
					if seq%int64(dropEveryN) == 0 {
						// Drop this packet — don't echo
						continue
					}
					pc.WriteTo(buf[:n], addr)
				}
			}
		}
	}()

	return port, func() { pc.Close() }
}

// startDuplicatingEchoListener echoes every packet twice, producing duplicates.
func startDuplicatingEchoListener(t *testing.T) (int, func()) {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := pc.LocalAddr().(*net.UDPAddr).Port

	go func() {
		buf := make([]byte, udpMaxPacketSize)
		for {
			n, addr, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			if n >= udpHeaderSize {
				magic := binary.BigEndian.Uint32(buf[0:4])
				if magic == udpMagic {
					pc.WriteTo(buf[:n], addr)
					pc.WriteTo(buf[:n], addr) // duplicate
				}
			}
		}
	}()

	return port, func() { pc.Close() }
}

// startReorderingEchoListener buffers two packets at a time and sends
// them back in reverse order, producing out-of-order replies.
func startReorderingEchoListener(t *testing.T) (int, func()) {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := pc.LocalAddr().(*net.UDPAddr).Port

	go func() {
		buf1 := make([]byte, udpMaxPacketSize)
		buf2 := make([]byte, udpMaxPacketSize)
		for {
			// Read first packet
			n1, addr1, err := pc.ReadFrom(buf1)
			if err != nil {
				return
			}
			// Read second packet
			n2, addr2, err := pc.ReadFrom(buf2)
			if err != nil {
				// Got one but not two — send the first anyway
				if n1 >= udpHeaderSize && binary.BigEndian.Uint32(buf1[0:4]) == udpMagic {
					pc.WriteTo(buf1[:n1], addr1)
				}
				return
			}
			// Send them in reverse order
			if n2 >= udpHeaderSize && binary.BigEndian.Uint32(buf2[0:4]) == udpMagic {
				pc.WriteTo(buf2[:n2], addr2)
			}
			if n1 >= udpHeaderSize && binary.BigEndian.Uint32(buf1[0:4]) == udpMagic {
				pc.WriteTo(buf1[:n1], addr1)
			}
		}
	}()

	return port, func() { pc.Close() }
}

func TestProbeUDP_NoLoss(t *testing.T) {
	port, cleanup := startTestEchoListener(t)
	defer cleanup()

	result := ProbeUDP("127.0.0.1", port, 10, 64, 2*time.Second)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.LossPct != 0 {
		t.Errorf("expected 0%% loss, got %.1f%%", result.LossPct)
	}
	if result.AvgRttS <= 0 {
		t.Errorf("expected positive RTT, got %.6f s", result.AvgRttS)
	}
	t.Logf("avg UDP RTT: %.4f ms", result.AvgRttS*1000)
}

func TestProbeUDP_FullLoss(t *testing.T) {
	// Bind a port then close it so nothing is listening.
	// This avoids assuming a hardcoded port like 19999 is free.
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := pc.LocalAddr().(*net.UDPAddr).Port
	pc.Close()

	result := ProbeUDP("127.0.0.1", port, 5, 64, 200*time.Millisecond)
	if result.LossPct != 100 {
		t.Errorf("expected 100%% loss, got %.1f%%", result.LossPct)
	}
}

func TestProbeUDP_PartialLoss(t *testing.T) {
	tests := []struct {
		name        string
		count       int
		dropEveryN  int
		expectedPct float64
	}{
		{"drop every 2nd (50%)", 10, 2, 50.0},
		{"drop every 3rd (33.3%)", 9, 3, 100.0 / 3.0},
		{"drop every 5th (20%)", 10, 5, 20.0},
		{"drop every 10th (10%)", 10, 10, 10.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, cleanup := startLossyEchoListener(t, tt.dropEveryN)
			defer cleanup()

			result := ProbeUDP("127.0.0.1", port, tt.count, 64, 2*time.Second)
			if result.Err != nil {
				t.Fatalf("unexpected error: %v", result.Err)
			}
			// Allow small floating point tolerance
			diff := result.LossPct - tt.expectedPct
			if diff < -0.1 || diff > 0.1 {
				t.Errorf("expected %.1f%% loss, got %.1f%%", tt.expectedPct, result.LossPct)
			}
			t.Logf("loss: %.1f%% (expected %.1f%%)", result.LossPct, tt.expectedPct)
		})
	}
}

func TestProbeUDP_ZeroCount(t *testing.T) {
	result := ProbeUDP("127.0.0.1", 12345, 0, 64, 200*time.Millisecond)
	if result.Err == nil {
		t.Error("expected error for count=0, got nil")
	}
}

func TestProbeUDP_PacketFormat(t *testing.T) {
	pkt := make([]byte, 64)
	binary.BigEndian.PutUint32(pkt[0:4], udpMagic)
	binary.BigEndian.PutUint32(pkt[4:8], 42)
	binary.BigEndian.PutUint64(pkt[8:16], uint64(time.Now().UnixNano()))

	magic := binary.BigEndian.Uint32(pkt[0:4])
	if magic != 0x47504E47 {
		t.Errorf("expected magic 0x47504E47, got 0x%X", magic)
	}
	seq := binary.BigEndian.Uint32(pkt[4:8])
	if seq != 42 {
		t.Errorf("expected seq 42, got %d", seq)
	}
}

func TestProbeUDP_Duplicates(t *testing.T) {
	port, cleanup := startDuplicatingEchoListener(t)
	defer cleanup()

	result := ProbeUDP("127.0.0.1", port, 5, 64, 2*time.Second)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.LossPct != 0 {
		t.Errorf("expected 0%% loss, got %.1f%%", result.LossPct)
	}
	if result.Duplicates == 0 {
		t.Error("expected duplicates > 0, got 0")
	}
	t.Logf("duplicates detected: %d", result.Duplicates)
}

func TestProbeUDP_OutOfOrder(t *testing.T) {
	port, cleanup := startReorderingEchoListener(t)
	defer cleanup()

	result := ProbeUDP("127.0.0.1", port, 10, 64, 2*time.Second)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.LossPct != 0 {
		t.Errorf("expected 0%% loss, got %.1f%%", result.LossPct)
	}
	if result.OutOfOrder == 0 {
		t.Error("expected out-of-order > 0, got 0")
	}
	t.Logf("out-of-order detected: %d, duplicates: %d", result.OutOfOrder, result.Duplicates)
}

func TestEstimateHops(t *testing.T) {
	tests := []struct {
		ttl  int
		want int32
	}{
		{64, 0},  // same host, Linux
		{63, 1},  // 1 hop, Linux
		{56, 8},  // 8 hops, Linux
		{128, 0}, // same host, Windows (TTL > 64 → initial=128)
		{127, 1}, // 1 hop, Windows
		{0, 0},   // invalid
	}
	for _, tt := range tests {
		got := estimateHops(tt.ttl)
		if got != tt.want {
			t.Errorf("estimateHops(%d) = %d, want %d", tt.ttl, got, tt.want)
		}
	}
}
