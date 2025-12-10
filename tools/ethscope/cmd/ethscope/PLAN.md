# Ethscope Buffering & UI Fidelity Plan

## Objectives
- **Ingest more samples than the current ~0.4 s ring** (1 000 000 entries ÷ 2.4 MSa/s) so the UI can freely scrub/divide up to the 100 ms/div range without starvation.
- **Decouple UDP ingest from WebSocket pushes** so transient stalls in the browser or trigger path never back up the capture flow.
- **Expose richer metadata to the UI** (accurate sample rate, trigger offsets, buffer fill, drop counters) so autoset/time/div controls can behave like a benchtop scope.

## Current State & Constraints
- `sampleBuffer` keeps at most `ringCapacityPerCh = 1_000_000` samples per channel, overwriting on overflow.
- Snapshotting always copies the newest `snapshotSamples` points, which limits the time base and prevents the UI from scrolling back.
- Trigger metadata is stored per packet; if a trigger is skipped (e.g., holdoff) we lose the context quickly because older packets are dropped.
- UDP ingest, trigger evaluation, ring-copying, and WS JSON marshalling all run inline inside `runUDPReceiver`, leaving no cushion when the PC is busy.
- Front-end rendering downsamples to `MAX_DISPLAY_POINTS = 2048` using simple nearest-neighbour indexing, so low-frequency square waves alias badly and duty cycles appear wrong despite clean MCU data.

## Proposed Architecture
1. **Two‑tier capture store**
   - Keep a raw sample ring sized in seconds (`historySeconds` → e.g., 20 s) allocated once at startup (`channels × sampleRate × historySeconds` entries, still <100 MB at 8‑bit).
   - Layer a decimated `overview` store (e.g., 1:32) used for coarse zooms and autoset; update it on ingest to avoid on-demand resampling.
2. **Asynchronous ingest pipeline**
   - UDP goroutine parses packets and enqueues `(header, samples)` objects into a bounded `chan`.
   - A worker goroutine drains the queue, runs trigger logic, writes into the rings, and updates stats. Backpressure only affects UDP reads when the queue is full.
3. **Snapshot API redraw**
   - Replace `snapshot(maxSamples)` with `snapshotWindow(spanSamples, anchor)` returning both raw (for fine zoom) and overview series plus exact `FirstSampleIdx`.
   - Add helper to compute decimated traces for UI’s `MAX_DISPLAY_POINTS = 2048` limit without re-walking the raw ring per frame.
4. **Metadata plumbing**
   - Track `drops`, `ingestLag`, `ringFillSamples`, and `triggerIndexAbs` inside `sampleBuffer`.
   - Include these fields in the WS payload so the UI can show buffer health and align the trigger marker even when displaying historical data.

## Implementation Steps
1. **Refactor buffering primitives**
   - Generalize `channelRing` to expose read cursors and random-access slices (avoid full copies for each snapshot).
   - Introduce `historyConfig` (seconds, sample rate, decimation ratio) and allocate rings accordingly.
2. **Introduce ingest worker**
   - Define `type captureJob struct { hdr packetHeader; samples [][]uint16; trig triggerInfo }`.
   - Make `runUDPReceiver` push jobs to `chan captureJob` and exit only on fatal error; move trigger evaluation + buffer writes into `captureLoop`.
3. **Snapshot redesign**
   - Implement `sampleBuffer.snapshotWindow(span time.Duration)` that chooses between raw/overview rings, packages `[]uint16` already downsampled to ≤2048 pts, and adjusts trigger indices.
   - Update `wsHub.broadcastLatest` to request a span derived from the UI’s current time/div plus guard bands.
4. **Expose stats & controls**
   - Extend `packetEvent` JSON with `HistorySeconds`, `BufferUtilization`, `DropCount`, `IngestDelayUs`, and `TriggerAbsIdx`.
   - Add WS command(s) for the browser to request different spans (e.g., autoset chooses 5 divisions around trigger).
5. **Alias-free decimation for UI**
   - Provide `downsampleMinMax`/`downsampleAverage` helpers in Go (or precomputed frame payload) so each pixel spans a time bucket with min/max envelopes.
   - Update the JS renderer to plot these envelopes (filled areas or thin lines) so 1 kHz square waves at 50 ms/div stay at 50 % duty visually.
6. **Validation & tooling**
   - Write a `capture_test.go` with synthetic packets to ensure drops, decimation, and snapshot math behave across wrap-arounds.
   - Add an optional `--ingest-q` CLI flag to size the queue, and a `--history` duration flag for easy tuning during bench work.

## Risks & Mitigations
- **Large allocations**: guard the history buffer with sanity checks (e.g., refuse >120 s unless the user overrides) and log actual memory use.
- **Latency spikes**: measure queue depth and emit warnings if UDP packets pile up; optionally drop the oldest job rather than blocking the network stack.
- **UI contract changes**: version the WS payload (add `schema_version`) so the frontend can feature-detect the richer data.
