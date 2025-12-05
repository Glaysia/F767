# ADC Ethernet Streaming Protocol

This document captures the committed plan for streaming ADC samples from the Nucleo-F767ZI to a PC. Firmware code (`EthStream` in `User/Inc/eth.hh`, `User/Src/eth.cc`) packetizes data and transmits it over UDP. Each channel intentionally carries redundancy: every sample is sent twice and each 256-sample block includes one parity byte, yielding ≈38.5 Mbit/s per channel (below the relaxed 50 Mbit/s ceiling) while providing lightweight error detection.

## Transport Layer
- **Transport:** IPv4 UDP via the LWIP raw API.
- **Source port:** Auto-assigned by LWIP.
- **Destination:** `192.168.10.1:5000`.
- **Cadence:** One packet per ADC DMA half-buffer. At 2.4 MSa/s with 256 samples/channel, packets are emitted at ≈4.7 kHz.

## Packet Format
UDP payloads contain a fixed header followed by original samples, their duplicates, and block parity bytes. All fields are little-endian.

```
Offset  Size  Field             Description
0x00    4     packet_seq        uint32_t, increments every packet (wraps allowed)
0x04    8     first_sample_idx  uint64_t, absolute index of the first *original* sample
0x0C    2     channels          uint16_t, active channel count (currently 1)
0x0E    2     samples_per_ch    uint16_t, original samples per channel (256)
0x10    2     flags             uint16_t, bit0=drop/overrun latched
0x12    2     sample_bits       uint16_t, bits per original sample (8)
0x14    N     payload           originals + duplicates + parity bytes
```

### Payload Structure
Per channel, the payload contains:
1. **Original samples:** 256 unsigned bytes. If multiple channels are enabled later, samples remain channel-interleaved.
2. **Duplicate samples:** The same 256 bytes repeated verbatim to provide immediate redundancy on the wire.
3. **Parity byte:** XOR of the 256 original samples (one byte per channel block) appended after the duplicate block.

When additional channels are enabled, their original/duplicate/parity blocks are concatenated in channel order.

## Operation
- TIM5 triggers ADC1 conversions; DMA fills a circular buffer sized for 256 samples per half-buffer.
- `HAL_ADC_ConvHalfCpltCallback` and `_CpltCallback` enqueue the start of each half-buffer.
- `AdcHandler::Process()` copies the half-buffer, constructs the payload (originals → duplicates → parity), populates the header, and calls `udp_send()`.
- On queue overflow or UDP failure, `flags` bit0 is set so the receiver can mark the packet unreliable.
- `first_sample_idx` increments by `samples_per_ch` after each packet, giving the host an absolute timeline.

## Bandwidth Commitment
- Original payload: 256 samples × 8 bits × 2.4 MSa/s → 19.2 Mbit/s.
- Duplication doubles payload → ≈38.4 Mbit/s.
- Parity adds 1 byte per 256 original samples → +0.39 % → ≈38.5 Mbit/s per channel.
- Two channels follow the same scheme, resulting in ≈77 Mbit/s aggregate while respecting 100 Mbit/s Ethernet limits.

## Receiver Expectations
- Validate `sample_bits`, `channels`, and `samples_per_ch` before parsing.
- Use the first 256-byte block as the canonical data, verify against the duplicate, and XOR-check the parity byte per block.
- Detect dropped packets via `packet_seq` gaps or `flags` bit0 and realign via `first_sample_idx`.

---

## 한글 설명

이 문서는 Nucleo-F767ZI에서 PC로 ADC 데이터를 전송하는 **확정 프로토콜**을 요약한다. 펌웨어(`EthStream`)는 UDP로 데이터를 보내며, 각 채널은 샘플을 두 번 전송하고 256샘플마다 패리티 1바이트를 추가해 채널당 약 38.5 Mbit/s(완화된 50 Mbit/s 한도 내)를 사용한다.

### 전송 계층
- **Transport:** IPv4 UDP (LWIP raw API)
- **Source port:** 자동 할당
- **Destination:** `192.168.10.1:5000`
- **주기:** ADC DMA 하프버퍼마다 1패킷 (256샘플/채널, 2.4 MSa/s ⇒ 약 4.7 kHz)

### 패킷 포맷
UDP 페이로드는 고정 헤더와 “원본 샘플 → 중복 샘플 → 패리티” 순서로 구성된다. 모든 필드는 리틀엔디언이다.

```
Offset  Size  Field                설명
0x00    4     packet_seq           uint32_t, 패킷마다 증가
0x04    8     first_sample_idx     uint64_t, 첫 원본 샘플의 절대 인덱스
0x0C    2     channels             uint16_t, 활성 채널 수 (현재 1)
0x0E    2     samples_per_ch       uint16_t, 원본 샘플 수 (256)
0x10    2     flags                uint16_t, bit0=드롭/오버런
0x12    2     sample_bits          uint16_t, 원본 샘플 비트 수 (8)
0x14    N     payload              원본/중복/패리티 데이터
```

#### 페이로드 구조
- 256바이트 원본 샘플(다채널 시 채널 인터리브)
- 동일한 256바이트를 그대로 중복
- 원본 256샘플 XOR 패리티 1바이트

채널이 늘어나면 채널 순서대로 위 블록을 반복해 붙인다.

### 동작 및 에러 처리
- TIM5→ADC1 DMA가 하프버퍼를 채우면 콜백에서 큐에 등록하고, `AdcHandler::Process()`가 헤더와 페이로드를 구성해 `udp_send()`를 호출한다.
- 큐 오버플로 또는 UDP 실패 시 `flags` bit0를 세트해 수신 측이 손상을 감지할 수 있게 한다.
- `first_sample_idx`는 패킷마다 256씩 증가해 호스트가 절대 시간축을 복원한다.

### 대역폭
- 원본 데이터 19.2 Mbit/s → 중복으로 38.4 Mbit/s.
- 256샘플/패리티 1바이트 = 0.39% 오버헤드 ⇒ 총 약 38.5 Mbit/s/채널.
- 2채널 사용 시 동일 구조로 ≈77 Mbit/s를 사용한다(100 Mbit/s 이더넷 내에서 동작).

### 수신기 요구 사항
- 헤더(`sample_bits`, `channels`, `samples_per_ch`)로 포맷을 검증한다.
- 첫 번째 블록을 기준 데이터로 삼고, 중복 및 패리티로 오류를 검사한다.
- `packet_seq`/`flags`로 드롭을 감지하고, `first_sample_idx`로 시간 정렬을 맞춘다.
