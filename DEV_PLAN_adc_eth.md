# ADC→Ethernet (UDP) implementation plan

## Targets (rev A, single-channel 8-bit)
- ADC1 single regular channel, 8-bit data, TIM5-triggered at max safe rate (PCLK2=72 MHz → ADC clk 36 MHz with DIV2).
- UDP stream ≥70 Mbps sustained without stalls; packet loss surfaced via flags/seq.
- Minimal CPU copy; DMA circular, raw UDP API, caching handled.

## Near-term steps
1) **ADC timing + dual path hook**
   - Keep ADC1 master single channel working; add scaffold to enable ADC2 in dual-regular simultaneous (interleave) later.
   - Ensure `ContinuousConvMode=DISABLE`, `ExternalTrigConvEdge=RISING`, prescaler DIV2 (36 MHz), sampling cycles at the minimum allowed for source impedance.
   - Retune TIM5 ARR/PSC for desired per-channel sample rate and verify DMA length matches (8-bit samples → halfword DMA still fine).

2) **DMA buffer & packet sizing**
   - Grow DMA half-buffer to ~`channel_count * 512–1024` samples so each UDP packet is 3–6 KB; reduces PPS and LWIP overhead.
   - Ensure buffer resides in SRAM1 (cacheable) and clean DCache for TX slices before send.

3) **TX path hardening**
   - Increase ETH descriptor counts (`ETH_TX_DESC_CNT/ETH_TXBUFNB/ETH_RXBUFNB` ~12) and LWIP heap/pbuf pool (`MEM_SIZE`, `PBUF_POOL_SIZE`) to survive high PPS.
   - Build pbuf chain with header (RAM) + payload (PBUF_REF to DMA window); on ERR_MEM set drop flag and move on.

4) **Bookkeeping & flags**
   - Track `packet_seq`, `first_sample_idx` (per-channel samples), `flags` bit0=drop/overrun.
   - Add basic stats counters (drops, sent) for UART debug.

5) **Dual ADC upgrade (next)**
   - Configure ADC2 slave, dual-regular simultaneous mode; ADC1 DMA reads interleaved samples.
   - Update packet header to report channel count=2 and adjust payload stride; revalidate sample index math.

## PC-side considerations
- Parser must handle variable `samples_per_ch` and flag gap detection via `packet_seq`.
- Capture tool: allow setting remote IP/port and saving raw payload for throughput testing.***
