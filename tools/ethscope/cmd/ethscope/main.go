package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
)

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>F767 Scope</title>
  <style>
    :root { color-scheme: light; font-family: "Segoe UI", Arial, sans-serif; background: #0b1c2c; color: #e8f0f7; }
    body { margin: 0; display: grid; place-items: center; height: 100vh; }
    .card { background: rgba(16, 38, 58, 0.9); border: 1px solid #264c6f; border-radius: 14px; padding: 20px 24px; width: min(560px, 92vw); box-shadow: 0 20px 50px rgba(0,0,0,0.35); }
    h1 { margin: 0 0 8px; font-size: 24px; letter-spacing: 0.4px; }
    p { margin: 4px 0; line-height: 1.5; }
    code { background: #0f2438; padding: 2px 6px; border-radius: 6px; }
    .status { margin-top: 12px; padding: 10px 12px; border-radius: 10px; background: rgba(255,255,255,0.04); border: 1px dashed #436f97; font-size: 14px; }
  </style>
</head>
<body>
  <div class="card">
    <h1>F767 이더넷 오실로스코프</h1>
    <p>Go 기반 수신기/웹서버가 동작 중입니다.</p>
    <p>다음 단계: UDP 패킷 파서, 링 버퍼, WebSocket 스트림을 추가해 실시간 파형을 전송하세요.</p>
    <div class="status">
      서버 상태: <strong id="status">준비 완료</strong><br/>
      WebSocket 엔드포인트: <code>/ws</code> (향후 구현 예정)
    </div>
  </div>
</body>
</html>`

const (
	protoHeaderSize = 0x14
	defaultPreview  = 8
)

type packetHeader struct {
	PacketSeq      uint32
	FirstSampleIdx uint64
	Channels       uint16
	SamplesPerCh   uint16
	Flags          uint16
	SampleBits     uint16
}

func main() {
	listen := flag.String("listen", ":8080", "HTTP listen address, e.g. :8080 or 0.0.0.0:8080")
	udpListen := flag.String("udp", ":5000", "UDP listen address for ADC packets")
	flag.Parse()

	go func() {
		if err := runUDPReceiver(*udpListen); err != nil {
			log.Fatalf("udp receiver stopped: %v", err)
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, indexHTML)
	})

	addr := *listen
	if strings.HasPrefix(addr, ":") {
		addr = "localhost" + addr
	}

	log.Printf("Serving placeholder UI at http://%s (UDP listener on %s)\n", addr, *udpListen)
	if err := http.ListenAndServe(*listen, mux); err != nil {
		log.Fatalf("http server failed: %v", err)
	}
}

func runUDPReceiver(listenAddr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return fmt.Errorf("resolve udp addr: %w", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("listen udp: %w", err)
	}
	defer conn.Close()

	log.Printf("UDP receiver listening on %s", conn.LocalAddr())
	buffer := make([]byte, 65536)

	for {
		n, remote, err := conn.ReadFromUDP(buffer)
		if err != nil {
			return fmt.Errorf("udp read: %w", err)
		}
		if n == 0 {
			continue
		}

		data := buffer[:n]
		header, payload, err := parsePacket(data)
		if err != nil {
			log.Printf("invalid packet from %s: %v", remote, err)
			continue
		}

		report, err := summarizePayload(header, payload)
		if err != nil {
			log.Printf("payload error from %s seq=%d: %v", remote, header.PacketSeq, err)
			continue
		}

		log.Println(report)
	}
}

func parsePacket(data []byte) (packetHeader, []byte, error) {
	if len(data) < protoHeaderSize {
		return packetHeader{}, nil, fmt.Errorf("packet too small: %d bytes", len(data))
	}

	h := packetHeader{
		PacketSeq:      binary.LittleEndian.Uint32(data[0:4]),
		FirstSampleIdx: binary.LittleEndian.Uint64(data[4:12]),
		Channels:       binary.LittleEndian.Uint16(data[12:14]),
		SamplesPerCh:   binary.LittleEndian.Uint16(data[14:16]),
		Flags:          binary.LittleEndian.Uint16(data[16:18]),
		SampleBits:     binary.LittleEndian.Uint16(data[18:20]),
	}

	payload := data[protoHeaderSize:]
	if h.Channels == 0 {
		return packetHeader{}, nil, errors.New("header reports zero channels")
	}
	if h.SampleBits == 0 || h.SampleBits%8 != 0 {
		return packetHeader{}, nil, fmt.Errorf("unsupported sample bits: %d", h.SampleBits)
	}

	sampleBytes := int(h.SampleBits / 8)
	perChannel := (int(h.SamplesPerCh) * sampleBytes * 2) + sampleBytes
	expected := perChannel * int(h.Channels)
	if len(payload) != expected {
		return packetHeader{}, nil, fmt.Errorf("payload mismatch: have %d, expected %d", len(payload), expected)
	}

	return h, payload, nil
}

func summarizePayload(h packetHeader, payload []byte) (string, error) {
	sampleBytes := int(h.SampleBits / 8)
	perChannel := (int(h.SamplesPerCh) * sampleBytes * 2) + sampleBytes
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("UDP seq=%d idx=%d ch=%d samples=%d flags=0x%X",
		h.PacketSeq, h.FirstSampleIdx, h.Channels, h.SamplesPerCh, h.Flags))

	for ch := 0; ch < int(h.Channels); ch++ {
		offset := ch * perChannel
		orig := payload[offset : offset+int(h.SamplesPerCh)*sampleBytes]
		dup := payload[offset+len(orig) : offset+len(orig)*2]
		parity := payload[offset+len(orig)*2 : offset+perChannel]

		if !bytes.Equal(orig, dup) {
			sb.WriteString(fmt.Sprintf(" [ch%d duplicate mismatch]", ch))
		}

		if err := verifyParity(orig, parity, sampleBytes); err != nil {
			sb.WriteString(fmt.Sprintf(" [ch%d parity %v]", ch, err))
		}

		preview := previewSamples(orig, sampleBytes, defaultPreview)
		sb.WriteString(fmt.Sprintf(" [ch%d first=%v]", ch, preview))
	}

	return sb.String(), nil
}

func verifyParity(samples, parity []byte, sampleBytes int) error {
	if len(parity) != sampleBytes {
		return fmt.Errorf("parity length %d != sample bytes %d", len(parity), sampleBytes)
	}

	sum := make([]byte, sampleBytes)
	for i := 0; i < len(samples); i += sampleBytes {
		for b := 0; b < sampleBytes; b++ {
			sum[b] ^= samples[i+b]
		}
	}

	for i := range parity {
		if sum[i] != parity[i] {
			return fmt.Errorf("expected % X got % X", sum, parity)
		}
	}
	return nil
}

func previewSamples(samples []byte, sampleBytes int, limit int) []uint32 {
	total := len(samples) / sampleBytes
	if limit > total {
		limit = total
	}
	out := make([]uint32, 0, limit)
	for i := 0; i < limit; i++ {
		start := i * sampleBytes
		out = append(out, decodeSample(samples[start:start+sampleBytes]))
	}
	return out
}

func decodeSample(b []byte) uint32 {
	var v uint32
	for i := 0; i < len(b); i++ {
		v |= uint32(uint8(b[i])) << (8 * i)
	}
	return v
}
