# Go Oscilloscope Receiver Plan

## 목표
- `NetworkProtocol.md`에 정의된 UDP 스트림을 수신해 패킷을 검증하고, 전송 중복/패리티를 활용해 오류를 감지한다.
- 수신 샘플을 큰 링 버퍼에 저장하고, 브라우저(PC/모바일)가 접근 가능한 HTTP/WebSocket 인터페이스를 통해 실시간 파형을 전송한다.
- 단일 Go 바이너리로 UDP 수신기와 웹 서버를 통합한다.

## 상위 아키텍처
1. **UDP 수신기 (`udp` 패키지)**  
   - `net.ListenUDP`로 포트 5000을 수신하고 패킷을 pre-allocated 슬라이스에 읽는다.  
   - 헤더를 파싱해 `sample_bits`, `channels`, `samples_per_ch`, `flags` 검증.  
   - 페이로드에서 원본/중복/패리티 블록을 분리해 즉시 CRC-like 검사를 수행한다.  
   - 유효 샘플을 `RingBuffer.Append(block)`로 넘긴다.  
   - 오류 발생 시 메트릭과 이벤트 채널로 보고한다.

2. **링 버퍼 (`buffer` 패키지)**  
   - 8비트 샘플 기준 최소 수 초 분량(예: 채널당 10만 샘플 × 채널수) 확보.  
   - `sync/atomic`으로 읽기/쓰기 인덱스를 관리해 Lock-free 멀티 프로듀서/컨슈머를 지원.  
   - WebSocket 브로드캐스터가 특정 시점부터 슬라이스를 읽을 수 있도록 `Snapshot(handle)` API 제공.  
   - 드롭/오버런 감지 시 통계 업데이트.

3. **웹 서버 (`server` 패키지)**  
   - `net/http` + `gorilla/websocket`으로 단순 HTTP 파일 서빙과 WebSocket 스트림 제공.  
   - `/` : 정적 HTML/JS/CSS (Canvas 기반 파형 뷰어).  
   - `/ws` : 링 버퍼에서 지속적으로 샘플을 꺼내 JSON 혹은 이진 메시지로 전송.  
   - 모바일 대응을 위해 적응형 레이트/뷰포트를 JS에서 제어.

4. **메트릭/상태 (`monitor` 패키지)**  
   - 최근 패킷 시퀀스, 드롭 플래그, 패리티 오류, 버퍼 사용률 등을 수집.  
   - `/status` JSON REST 엔드포인트와 WebSocket 제어 프레임에서 노출.

## 주요 타입 초안
```go
type PacketHeader struct {
    PacketSeq      uint32
    FirstSampleIdx uint64
    Channels       uint16
    SamplesPerCh   uint16
    Flags          uint16
    SampleBits     uint16
}

type RingBuffer struct {
    data   []byte
    head   uint64
    tail   uint64
    size   uint64
}
```

## 초기 구현 단계
1. `cmd/ethscope/main.go` 생성 후 플래그(UDP 포트, 웹 포트, 버퍼 길이) 파싱.
2. `udp` 패키지에서 수신 루프 작성, 헤더/페이로드 파서 구현.
3. `buffer` 패키지로 고정 길이 링 버퍼 추가 및 단위 테스트 작성.
4. `server` 패키지에서 HTTP 정적 서빙 + WebSocket 라우트 구성.
5. 간단한 JS 프런트엔드(`tools/web/`)로 파형을 Canvas에 그리는 MVP 작성.
6. 통합 테스트: 가짜 패킷 제너레이터로 UDP를 주입하고 브라우저에서 실시간 파형 확인.

## 추후 개선 아이디어
- 다중 채널 지원 시 WebSocket 메시지에 채널 ID를 포함하고 렌더러 레이아웃 추가.
- WebAssembly 기반 DSP(예: FIR 필터, 트리거 감지) 적용 옵션.
- Prometheus 메트릭 노출 및 pprof 기반 성능 모니터링.
