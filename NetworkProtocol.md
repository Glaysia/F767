# ADC Ethernet Streaming Protocol

This document describes the on-wire format implemented by `EthStream` (`User/Inc/eth.hh`, `User/Src/eth.cc`). The firmware sends 8-bit samples without duplication to keep bandwidth within the 100 Mbit/s Ethernet limit while maintaining low latency.

## Transport Layer
- **Transport:** IPv4 UDP via the LWIP raw API.
- **Source port:** Auto-assigned by LWIP.
- **Destination:** `192.168.10.1:5000`.
- **Cadence:** One packet per ADC DMA half-buffer. With 256 samples/channel at 2.4 MSa/s, packets are emitted at ≈4.7 kHz.

## Packet Format
UDP payloads contain a fixed header followed by raw interleaved sample bytes. All fields are little-endian.

```
Offset  Size  Field             Description
0x00    4     packet_seq        uint32_t, increments every packet (wraps allowed)
0x04    8     first_sample_idx  uint64_t, absolute index of the first sample in this packet
0x0C    2     channels          uint16_t, active channel count (currently 1)
0x0E    2     samples_per_ch    uint16_t, samples per channel in the payload (256 today)
0x10    2     flags             uint16_t, bit0=drop/overrun latched
0x12    2     sample_bits       uint16_t, bits per sample (8)
0x14    N     payload           channel-interleaved raw samples (1 byte/sample)
```

### Payload
- Samples are interleaved by channel: `[ch0_s0][ch1_s0][ch0_s1][ch1_s1]…`.
- With a single channel active, the payload is a flat 256-byte block.

## Operation
- TIM5 drives ADC DMA in circular mode. Callbacks (`HAL_ADC_ConvHalfCpltCallback` / `HAL_ADC_ConvCpltCallback`) enqueue each half-buffer into a byte queue (512-frame depth ≈54 ms at 2.4 MSa/s/channel).
- `AdcHandler::Process()` runs in the main loop (right after `MX_LWIP_Process()`), drains the queue, fills the header, and calls `udp_send()`.
- On queue overflow or UDP failure, `flags` bit0 is latched and reported with the next packet.
- `first_sample_idx` is captured per DMA half-buffer so the host timeline stays monotonic even if a frame is dropped.

## Bandwidth
- One channel: 256 samples × 8 bits × 2.4 MSa/s → ≈19.2 Mbit/s payload.
- Two channels (future) at the same rate: ≈38.4 Mbit/s aggregate, leaving headroom under 100 Mbit/s Ethernet.

## Receiver Notes
- Validate `sample_bits`, `channels`, and `samples_per_ch` before parsing.
- Use `packet_seq` gaps or `flags` bit0 to detect drops; align time using `first_sample_idx`.
- Samples are unsigned bytes; scale to volts as needed in the host application.

---

## 한글 설명

이 문서는 펌웨어(`EthStream`)가 사용하는 UDP 패킷 포맷을 설명한다. 샘플은 8비트로 전송되며, 중복이나 패리티 없이 헤더 뒤에 바로 인터리브된 원본 바이트가 붙는다.

### 전송 계층
- **Transport:** IPv4 UDP (LWIP raw API)
- **Destination:** `192.168.10.1:5000`
- **주기:** DMA 하프버퍼마다 1패킷 (256샘플/채널, 2.4 MSa/s ⇒ 약 4.7 kHz)

### 패킷 포맷
- 헤더 필드: `packet_seq`, `first_sample_idx`, `channels`, `samples_per_ch`, `flags(bit0=드롭)`, `sample_bits(8)`.
- 페이로드: 채널 인터리브된 1바이트 샘플 배열. 단일 채널일 경우 256바이트.

### 동작 및 에러 처리
- TIM5→ADC DMA 콜백에서 하프버퍼를 8비트 큐(512프레임)로 복사하고, 메인 루프의 `AdcHandler::Process()`가 헤더를 채워 `udp_send()`를 호출한다.
- 큐 오버플로 또는 UDP 실패 시 `flags` bit0을 세트해 다음 패킷에 보고한다.
- `first_sample_idx`는 DMA 하프버퍼 캡처 시 기록되어 드롭이 있어도 시간축이 단조롭게 유지된다.

### 대역폭
- 1채널 8비트 2.4 MSa/s → 약 19.2 Mbit/s.
- 2채널 확장 시 ≈38.4 Mbit/s로 100 Mbit/s 이더넷 한도 내 여유가 있다.

## Function Generator Relay
- **Transport:** UDP ASCII control.
- **Destination:** MCU1 `192.168.10.2:6001` (default, overridable via `--fg-addr` on the host UI).
- **Payload:** UART.md 명령 문자열을 그대로 보냄 (`F1000`, `A2048`, `W0`, `D`, `H` 등). MCU1이 UART2(115200/8/N/1, TX=PD5, RX=PA3)로 전달한다.
