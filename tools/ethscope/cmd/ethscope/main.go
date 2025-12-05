package main

import (
	"flag"
	"fmt"
	"log"
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

func main() {
	listen := flag.String("listen", ":8080", "HTTP listen address, e.g. :8080 or 0.0.0.0:8080")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, indexHTML)
	})

	addr := *listen
	if strings.HasPrefix(addr, ":") {
		addr = "localhost" + addr
	}

	log.Printf("Serving placeholder UI at http://%s\n", addr)
	if err := http.ListenAndServe(*listen, mux); err != nil {
		log.Fatalf("http server failed: %v", err)
	}
}
