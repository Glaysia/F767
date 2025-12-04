# ADC→Ethernet (UDP) implementation plan

## Targets
- Stream one channel on ADC1 and one channel on ADC2 (e.g., ADC1_IN0 and ADC2_IN3) at 8-bit resolution around 2.4 MSa/s per channel (≈19.2 Mbit/s payload each) with minimal CPU copy so every stream remains under the 35 Mbit/s budget.
- Use TIM5-paced dual-ADC DMA (circular) and the LWIP raw UDP API while preserving timing info so the PC GUI can align waveforms and absorb bursts into host memory.

## Data model
- Frame: one DMA half-buffer block = `2 ADCs × 1 channel × 512 samples = 1024 bytes` (8-bit samples). At 2.4 MSa/s per ADC this block covers 512 / 2.4e6 ≈ 213 µs, implying a processing cadence of ~4.69 kHz per DMA half-transfer.
- Packet: header + one frame (aggregate multiple frames later if that helps NIC interrupt rate).
- Header (per packet):
  - `uint32_t packet_seq` (monotonic, wrap ok)
  - `uint64_t first_sample_index` (trigger count since boot; increment by samples-per-channel per packet)
  - `uint16_t channels = 2`
  - `uint16_t samples_per_ch = 512`
  - `uint16_t sample_bits = 8` (for GUI sanity)
  - `uint16_t flags` (bit0=overrun/drop)

## Firmware tasks
1) **ADC DMA callbacks**
   - Implement `HAL_ADC_ConvHalfCpltCallback` / `HAL_ADC_ConvCpltCallback` in C (e.g., `User/Src/adc_dma.c`).
   - Each callback computes `base_ptr` (front/back half) for the interleaved ADC1/ADC2 buffer and calls a queue function with pointer + sample count.

2) **TX queue / zero-copy**
   - Define a small ring of TX descriptors:
     - header struct (above)
     - payload pointer to ADC buffer region
     - length bytes (= header + 2 × 512 bytes of payload)
   - If LWIP pbuf supports PBUF_REF chain, attach header (PBUF_RAM) + data (PBUF_REF to ADC buffer half). Otherwise, copy data into a TX scratch buffer (worst case).
   - On queue full, drop oldest/newest and set `flags|=1` in header.

3) **LWIP init & socket**
   - In `lwip.c` post-init hook, create a UDP PCB, set remote IP/port (hardcoded or from config), and store a handle.
   - Provide `NetSendPbuf(const struct pbuf *p)` wrapper that returns ERR_OK/ERR_MEM.

4) **Integration path**
   - Add a C API callable from callbacks: `void AdcEth_OnFrame(uint16_t *base, size_t samples)`.
   - Inside, build header (packet_seq++, first_sample_index tracked in `uint64_t`), build pbuf chain, call UDP send.
   - After send, free only the header pbuf; ADC buffer is owned by DMA, so do not free that segment.

5) **Timing/accounting**
   - Maintain `uint64_t g_sample_index` incremented by `samples_per_ch` each frame (i.e., +512 per packet with the current plan).
   - Optionally latch a free-running timer value (TIM2 or DWT) into header if GUI needs finer sync.

6) **Error handling**
   - If UDP send fails (ERR_MEM), set drop flag and skip; do not block callbacks.
   - Consider a lightweight stats struct (drops, sent packets) exposed via UART/log for debug.

## PC-side assumptions
- Sample clock tuned so each channel delivers ~2.4 MSa/s (8-bit) which equals ≈19.2 Mbit/s payload per channel, leaving headroom under the 35 Mbit/s per-channel budget.
- Packet loss handled via `packet_seq` gap; timeline via `first_sample_index / trigger_rate`.
- Host app must allocate sufficient buffering to absorb the ~38–40 Mbit/s aggregate UDP stream before rendering to the screen.

## Optional later
- Aggregate two frames per packet to reduce overhead if bandwidth headroom allows.
- Add runtime-configurable remote IP/port via UART command.
