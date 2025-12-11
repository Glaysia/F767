# MCU Capture Drop Investigation Plan

## Findings (current code)
- ADC DMA runs at TIM5 ARR=29 (≈3.6 MSa/s) with 256-sample blocks (`kAdcDmaSamplesPerChannel`). This produces ~14 k blocks/sec → ~14 k UDP packets/sec, which is likely beyond lwIP + main-loop throughput and can drop frames before they reach UDP.
- `AdcHandler_Enqueue` drops a block silently when the queue is full (`next_write == g_frame_read`) or when `EthStream::SendFrame` fails; it only latches a flag. There is no UART/log output, and the PC viewer ignores `flags`, so pre-UDP drops are invisible.
- `first_sample_idx` advances even when a block is dropped, so gaps are detectable host-side only if the UDP packet is produced. If the drop happens before queuing/sending, the host sees no index gap, matching the observed “no warning”.

## Fix/Instrumentation plan
- **Slow the ADC to the target rate**: set TIM5 ARR back to 85 (≈1.256 MSa/s) to cut block rate to ~4.9 k packets/sec, matching the PC assumption and easing lwIP pressure.
- **Expose pre-UDP drops**:
  - Add counters for (a) queue overflow, (b) pbuf alloc failure / `udp_send` failure, and (c) ETH not ready. Stamp these into `EthPacketHeader.flags` as bits and periodically print over UART when non-zero.
  - When a block is dropped before enqueue/send, also advance a monotonic `frame_seq` and include it in the header so the PC can detect missing frames even without index gaps.
- **Watchdog for backlog**: track `g_frame_write - g_frame_read` depth; if it stays near `kAdcFrameQueueDepth`, emit a UART warning and optionally drop older frames aggressively to avoid long latency.
- **Validate in Go UI**: surface `flags` and new `frame_seq`/drop counters in the websocket payload and log warnings when non-zero to correlate with trigger instability.
- **Optional**: lower interrupt load by using a pointer swap instead of memcpy inside the ADC callbacks, moving the copy into the main loop, to reduce ISR time at high sample rates.
