// Copyright 2018 Bloomberg Finance L.P.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package goldpinger

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"time"

	"go.uber.org/zap"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	udpMagic      = 0x47504E47 // "GPNG"
	udpHeaderSize = 16         // 4 magic + 4 seq + 8 timestamp

	// udpMaxPacketSize is the maximum UDP packet we will read. Sized to the
	// standard Ethernet MTU (1500 bytes) which is the largest packet that
	// will survive most networks without fragmentation. Any configured
	// UDP_PACKET_SIZE larger than this will be clamped to this value to
	// avoid silent truncation on the receive side.
	udpMaxPacketSize = 1500
)

// UDPProbeResult holds the results of a UDP probe to a peer
type UDPProbeResult struct {
	LossPct  float64
	HopCount int32
	AvgRttS  float64
	Err      error
}

// StartUDPListener starts a UDP echo listener on the given port.
// It echoes back any packet that starts with the GPNG magic number.
func StartUDPListener(port int) {
	addr := net.JoinHostPort("::", strconv.Itoa(port))
	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		zap.L().Fatal("Failed to start UDP listener", zap.String("addr", addr), zap.Error(err))
	}
	defer pc.Close()

	zap.L().Info("UDP echo listener started", zap.String("addr", addr))

	buf := make([]byte, udpMaxPacketSize)
	for {
		n, remoteAddr, err := pc.ReadFrom(buf)
		if err != nil {
			zap.L().Warn("UDP read error", zap.Error(err))
			continue
		}
		if n < udpHeaderSize {
			continue
		}
		magic := binary.BigEndian.Uint32(buf[0:4])
		if magic != udpMagic {
			continue
		}
		// Echo back the packet as-is
		_, err = pc.WriteTo(buf[:n], remoteAddr)
		if err != nil {
			zap.L().Warn("UDP write error", zap.Error(err))
		}
	}
}

// ProbeUDP sends count UDP packets to the target and measures loss and hop count.
func ProbeUDP(targetIP string, port, count, size int, timeout time.Duration) UDPProbeResult {
	if count <= 0 {
		return UDPProbeResult{Err: fmt.Errorf("packet count must be > 0, got %d", count)}
	}
	if size < udpHeaderSize {
		size = udpHeaderSize
	}
	if size > udpMaxPacketSize {
		size = udpMaxPacketSize
	}

	addr := net.JoinHostPort(targetIP, strconv.Itoa(port))
	conn, err := net.Dial("udp", addr)
	if err != nil {
		CountUDPError(targetIP)
		return UDPProbeResult{LossPct: 100, Err: fmt.Errorf("dial: %w", err)}
	}
	defer conn.Close()

	// Determine if this is IPv4 or IPv6 and set up TTL/HopLimit reading
	isIPv6 := net.ParseIP(targetIP).To4() == nil
	var ttlValue int
	ttlFound := false

	if isIPv6 {
		p := ipv6.NewPacketConn(conn.(*net.UDPConn))
		if err := p.SetControlMessage(ipv6.FlagHopLimit, true); err != nil {
			zap.L().Debug("Cannot set IPv6 hop limit flag", zap.Error(err))
		}
	} else {
		p := ipv4.NewPacketConn(conn.(*net.UDPConn))
		if err := p.SetControlMessage(ipv4.FlagTTL, true); err != nil {
			zap.L().Debug("Cannot set IPv4 TTL flag", zap.Error(err))
		}
	}

	// Build packet
	pkt := make([]byte, size)
	binary.BigEndian.PutUint32(pkt[0:4], udpMagic)

	// Send all packets
	for i := 0; i < count; i++ {
		binary.BigEndian.PutUint32(pkt[4:8], uint32(i))
		binary.BigEndian.PutUint64(pkt[8:16], uint64(time.Now().UnixNano()))
		_, err := conn.Write(pkt)
		if err != nil {
			CountUDPError(targetIP)
			zap.L().Debug("UDP send error", zap.Int("seq", i), zap.Error(err))
		}
	}

	// Receive replies
	received := 0
	var totalRttNs int64
	deadline := time.Now().Add(timeout)
	conn.SetReadDeadline(deadline)

	recvBuf := make([]byte, udpMaxPacketSize)

	if isIPv6 {
		p := ipv6.NewPacketConn(conn.(*net.UDPConn))
		for received < count {
			n, cm, _, err := p.ReadFrom(recvBuf)
			now := time.Now()
			if err != nil {
				break
			}
			if n < udpHeaderSize {
				continue
			}
			magic := binary.BigEndian.Uint32(recvBuf[0:4])
			if magic != udpMagic {
				continue
			}
			sentNs := int64(binary.BigEndian.Uint64(recvBuf[8:16]))
			totalRttNs += now.UnixNano() - sentNs
			received++
			if cm != nil && cm.HopLimit > 0 && !ttlFound {
				ttlValue = cm.HopLimit
				ttlFound = true
			}
		}
	} else {
		p := ipv4.NewPacketConn(conn.(*net.UDPConn))
		for received < count {
			n, cm, _, err := p.ReadFrom(recvBuf)
			now := time.Now()
			if err != nil {
				break
			}
			if n < udpHeaderSize {
				continue
			}
			magic := binary.BigEndian.Uint32(recvBuf[0:4])
			if magic != udpMagic {
				continue
			}
			sentNs := int64(binary.BigEndian.Uint64(recvBuf[8:16]))
			totalRttNs += now.UnixNano() - sentNs
			received++
			if cm != nil && cm.TTL > 0 && !ttlFound {
				ttlValue = cm.TTL
				ttlFound = true
			}
		}
	}

	lossPct := float64(count-received) / float64(count) * 100.0

	var hopCount int32
	if ttlFound {
		hopCount = estimateHops(ttlValue)
	}

	var avgRttS float64
	if received > 0 {
		avgRttS = float64(totalRttNs) / float64(received) / 1e9
	}

	return UDPProbeResult{
		LossPct:  lossPct,
		HopCount: hopCount,
		AvgRttS:  avgRttS,
	}
}

// estimateHops estimates the number of hops from the received TTL.
// Common initial TTL values are 64 (Linux) and 128 (Windows).
func estimateHops(receivedTTL int) int32 {
	if receivedTTL <= 0 {
		return 0
	}
	initialTTL := 64
	if receivedTTL > 64 {
		initialTTL = 128
	}
	hops := initialTTL - receivedTTL
	if hops < 0 {
		hops = 0
	}
	return int32(hops)
}
