package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const indexHTML = `<!DOCTYPE html>
<html lang="ko">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>F767 Scope</title>
  <style>
    :root { color-scheme: dark; font-family: "Segoe UI", "Pretendard", sans-serif; background: #071323; color: #e5f0ff; }
    body { margin: 0; display: grid; place-items: center; min-height: 100vh; background: radial-gradient(circle at top, rgba(36,74,104,0.6), #050c16 60%); }
    .card { background: rgba(8, 20, 35, 0.95); border: 1px solid rgba(73, 123, 177, 0.4); border-radius: 18px; padding: 24px 28px 32px; width: min(1080px, 92vw); box-shadow: 0 30px 80px rgba(0,0,0,0.45); }
    h1 { margin: 0 0 4px; font-size: clamp(24px, 3vw, 32px); letter-spacing: 0.6px; }
    p { margin: 6px 0 18px; line-height: 1.4; opacity: 0.85; }
    canvas { width: 100%; max-height: 420px; border-radius: 16px; background: #030a14; border: 1px solid rgba(255,255,255,0.04); margin-top: 18px; box-shadow: 0 10px 40px rgba(0,0,0,0.35); }
    code { background: #0a1d2e; padding: 2px 6px; border-radius: 6px; }
    .status { display: flex; flex-wrap: wrap; gap: 16px; padding: 12px 14px; border-radius: 12px; background: rgba(255,255,255,0.03); border: 1px dashed rgba(90,140,200,0.6); font-size: 14px; }
    .hint { font-size: 13px; opacity: 0.7; margin-top: 10px; }
  </style>
</head>
<body>
  <div class="card">
    <h1>F767 이더넷 오실로스코프</h1>
    <p>Go 기반 수신기/웹서버가 UDP 데이터를 받아 WebSocket으로 브라우저 캔버스에 즉시 그립니다.</p>
    <div class="status">
      <div>서버 상태: <strong id="status">WebSocket 연결 대기</strong></div>
      <div>표시 샘플: 채널 0, 256샘플 블록 기준</div>
    </div>
    <canvas id="scope" width="960" height="360"></canvas>
    <p class="hint">UDP 스트림이 들어오면 WebSocket "/ws"가 샘플을 전송하고, 캔버스가 실시간 파형을 그립니다.</p>
  </div>
  <script>
  (function() {
    const statusEl = document.getElementById('status');
    const canvas = document.getElementById('scope');
    const ctx = canvas.getContext('2d');
    let reconnectTimer = null;

    function setStatus(text) {
      statusEl.textContent = text;
    }

    function drawWave(samples, bits) {
      if (!samples || !samples.length) {
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        return;
      }
      const max = Math.pow(2, bits) - 1 || 255;
      ctx.fillStyle = '#020611';
      ctx.fillRect(0, 0, canvas.width, canvas.height);
      ctx.strokeStyle = '#39e6ff';
      ctx.lineWidth = 2;
      ctx.beginPath();
      for (let x = 0; x < canvas.width; x++) {
        const idx = Math.floor(x / canvas.width * samples.length);
        const val = samples[idx] / max;
        const y = canvas.height - val * canvas.height;
        if (x === 0) ctx.moveTo(x, y);
        else ctx.lineTo(x, y);
      }
      ctx.stroke();
    }

    function connect() {
      const url = (window.location.protocol === 'https:' ? 'wss://' : 'ws://') +
                  window.location.host + '/ws';
      const ws = new WebSocket(url);

      ws.onopen = () => {
        setStatus('WebSocket 연결됨 · 샘플 대기 중');
        if (reconnectTimer) {
          clearTimeout(reconnectTimer);
          reconnectTimer = null;
        }
      };

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.samples && msg.samples.length) {
            drawWave(msg.samples[0], msg.sample_bits || 8);
            setStatus('seq ' + msg.seq + ' · ' + msg.samples_per_ch + ' samples');
          }
        } catch (err) {
          console.error('invalid ws message', err);
        }
      };

      ws.onclose = () => {
        setStatus('연결 끊김 · 재연결 시도 중…');
        reconnectTimer = setTimeout(connect, 1500);
      };

      ws.onerror = () => {
        ws.close();
      };
    }

    connect();
  })();
  </script>
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

type packetEvent struct {
	Seq            uint32     `json:"seq"`
	FirstSampleIdx uint64     `json:"first_idx"`
	Channels       uint16     `json:"channels"`
	SamplesPerCh   uint16     `json:"samples_per_ch"`
	SampleBits     uint16     `json:"sample_bits"`
	Flags          uint16     `json:"flags"`
	Samples        [][]uint16 `json:"samples"`
}

type packetStore struct {
	mu      sync.RWMutex
	version uint64
	seq     uint32
	data    []byte
}

func (ps *packetStore) Store(seq uint32, data []byte) {
	ps.mu.Lock()
	ps.seq = seq
	ps.data = data
	ps.version++
	ps.mu.Unlock()
}

func (ps *packetStore) Load() (uint64, uint32, []byte) {
	ps.mu.RLock()
	v := ps.version
	seq := ps.seq
	data := ps.data
	ps.mu.RUnlock()
	return v, seq, data
}

type wsHub struct {
	mu            sync.Mutex
	clients       map[*websocket.Conn]struct{}
	upgrader      websocket.Upgrader
	frameInterval time.Duration
	writeTimeout  time.Duration
	latest        packetStore
	lastBroadcast atomic.Uint64
}

func newWSHub(fps int) *wsHub {
	if fps <= 0 {
		fps = 30
	}
	if fps > 240 {
		fps = 240
	}
	interval := time.Second / time.Duration(fps)
	if interval <= 0 {
		interval = time.Second / 60
	}
	return &wsHub{
		clients: make(map[*websocket.Conn]struct{}),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		frameInterval: interval,
		writeTimeout:  500 * time.Millisecond,
	}
}

func (h *wsHub) Start() {
	go h.dispatchLoop()
}

func (h *wsHub) dispatchLoop() {
	ticker := time.NewTicker(h.frameInterval)
	defer ticker.Stop()
	for range ticker.C {
		h.broadcastLatest()
	}
}

func main() {
	listen := flag.String("listen", ":8080", "HTTP listen address, e.g. :8080 or 0.0.0.0:8080")
	udpListen := flag.String("udp", ":5000", "UDP listen address for ADC packets")
	dumpPackets := flag.Bool("dump-packets", false, "log each UDP packet summary to stdout")
	uiFPS := flag.Int("ui-fps", 60, "maximum WebSocket frame rate (frames per second)")
	flag.Parse()

	hub := newWSHub(*uiFPS)
	hub.Start()

	go func() {
		if err := runUDPReceiver(*udpListen, *dumpPackets, hub); err != nil {
			log.Fatalf("udp receiver stopped: %v", err)
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, indexHTML)
	})
	mux.HandleFunc("/ws", hub.handleWS)

	addr := *listen
	if strings.HasPrefix(addr, ":") {
		addr = "localhost" + addr
	}

	log.Printf("Serving UI at http://%s (UDP listener on %s, dump_packets=%v)\n", addr, *udpListen, *dumpPackets)
	if err := http.ListenAndServe(*listen, mux); err != nil {
		log.Fatalf("http server failed: %v", err)
	}
}

func runUDPReceiver(listenAddr string, dumpPackets bool, hub *wsHub) error {
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

		if dumpPackets {
			log.Println(report)
		}

		if err := hub.BroadcastPacket(header, payload); err != nil {
			log.Printf("ws broadcast error seq=%d: %v", header.PacketSeq, err)
		}
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

func buildPacketEvent(h packetHeader, payload []byte) ([]byte, error) {
	sampleBytes := int(h.SampleBits / 8)
	if sampleBytes <= 0 || sampleBytes > 2 {
		return nil, fmt.Errorf("sample bits %d not supported for UI", h.SampleBits)
	}

	perChannel := (int(h.SamplesPerCh) * sampleBytes * 2) + sampleBytes
	evt := packetEvent{
		Seq:            h.PacketSeq,
		FirstSampleIdx: h.FirstSampleIdx,
		Channels:       h.Channels,
		SamplesPerCh:   h.SamplesPerCh,
		SampleBits:     h.SampleBits,
		Flags:          h.Flags,
		Samples:        make([][]uint16, h.Channels),
	}

	for ch := 0; ch < int(h.Channels); ch++ {
		offset := ch * perChannel
		orig := payload[offset : offset+int(h.SamplesPerCh)*sampleBytes]
		ints := make([]uint16, h.SamplesPerCh)
		for i := 0; i < int(h.SamplesPerCh); i++ {
			start := i * sampleBytes
			ints[i] = uint16(decodeSample(orig[start : start+sampleBytes]))
		}
		evt.Samples[ch] = ints
	}

	return json.Marshal(evt)
}

func (h *wsHub) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade failed: %v", err)
		return
	}
	h.register(conn)
	go h.readPump(conn)
}

func (h *wsHub) register(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = struct{}{}
	total := len(h.clients)
	h.mu.Unlock()
	log.Printf("ws client connected (%d total)", total)
}

func (h *wsHub) remove(conn *websocket.Conn) {
	h.mu.Lock()
	if _, ok := h.clients[conn]; ok {
		delete(h.clients, conn)
	}
	total := len(h.clients)
	h.mu.Unlock()
	conn.Close()
	log.Printf("ws client disconnected (%d total)", total)
}

func (h *wsHub) readPump(conn *websocket.Conn) {
	defer h.remove(conn)
	conn.SetReadLimit(1024)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("ws read error: %v", err)
			}
			break
		}
	}
}

func (h *wsHub) BroadcastPacket(header packetHeader, payload []byte) error {
	if h == nil {
		return nil
	}

	msg, err := buildPacketEvent(header, payload)
	if err != nil {
		return err
	}

	h.latest.Store(header.PacketSeq, msg)
	return nil
}

func (h *wsHub) broadcastLatest() {
	version, _, data := h.latest.Load()
	if data == nil || version == 0 {
		return
	}
	if version == h.lastBroadcast.Load() {
		return
	}

	h.mu.Lock()
	if len(h.clients) == 0 {
		h.mu.Unlock()
		return
	}
	conns := make([]*websocket.Conn, 0, len(h.clients))
	for conn := range h.clients {
		conns = append(conns, conn)
	}
	h.mu.Unlock()

	h.lastBroadcast.Store(version)
	for _, conn := range conns {
		conn.SetWriteDeadline(time.Now().Add(h.writeTimeout))
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			h.remove(conn)
			log.Printf("ws write error: %v", err)
		}
	}
}
