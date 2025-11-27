# ADC→Ethernet (UDP) implementation plan

## Targets
- Stream ADC1 scan (CH0/CH3/CH4) over UDP with minimal CPU copy.
- Use TIM5-paced ADC DMA (circular) and LWIP raw UDP API.
- Preserve timing info so PC GUI can align waveforms.

## Data model
- Frame: one DMA half-buffer block = `3 channels × 64 samples = 192 halfwords`.
- Packet: header + one frame (no aggregation initially).
- Header (per packet):
  - `uint32_t packet_seq` (monotonic, wrap ok)
  - `uint64_t first_sample_index` (trigger count since boot; increment by 64 per packet)
  - `uint16_t channels = 3`
  - `uint16_t samples_per_ch = 64`
  - `uint16_t sample_bits = 12` (for GUI sanity)
  - `uint16_t flags` (bit0=overrun/drop)

## Firmware tasks
1) **ADC DMA callbacks**
   - Implement `HAL_ADC_ConvHalfCpltCallback` / `HAL_ADC_ConvCpltCallback` in C (e.g., `User/Src/adc_dma.c`).
   - Each callback computes `base_ptr` (front/back half) and calls a queue function with pointer + sample count.

2) **TX queue / zero-copy**
   - Define a small ring of TX descriptors:
     - header struct (above)
     - payload pointer to ADC buffer region
     - length bytes (= header + 192 * 2)
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
   - Maintain `uint64_t g_sample_index` incremented by `samples_per_ch` each frame (i.e., +64 per packet).
   - Optionally latch a free-running timer value (TIM2 or DWT) into header if GUI needs finer sync.

6) **Error handling**
   - If UDP send fails (ERR_MEM), set drop flag and skip; do not block callbacks.
   - Consider a lightweight stats struct (drops, sent packets) exposed via UART/log for debug.

## PC-side assumptions
- Sample rate known: TIM5 update ~1.2558 MHz trigger → per-channel ~418.6 kS/s.
- Packet loss handled via `packet_seq` gap; timeline via `first_sample_index / trigger_rate`.

## Optional later
- Aggregate two frames per packet to reduce overhead if bandwidth headroom allows.
- Add runtime-configurable remote IP/port via UART command.
