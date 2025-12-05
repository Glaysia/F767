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
    .card { background: rgba(8, 20, 35, 0.95); border: 1px solid rgba(73, 123, 177, 0.4); border-radius: 18px; padding: 24px 28px 32px; width: min(1100px, 96vw); box-shadow: 0 30px 80px rgba(0,0,0,0.45); }
    h1 { margin: 0 0 6px; font-size: clamp(24px, 3vw, 32px); letter-spacing: 0.6px; }
    p { margin: 4px 0 14px; line-height: 1.5; opacity: 0.85; }
    canvas { width: 100%; max-height: 420px; border-radius: 16px; background: #030a14; border: 1px solid rgba(255,255,255,0.04); margin-top: 18px; box-shadow: 0 10px 40px rgba(0,0,0,0.35); }
    .status { display: flex; flex-wrap: wrap; gap: 16px; padding: 12px 14px; border-radius: 12px; background: rgba(255,255,255,0.03); border: 1px dashed rgba(90,140,200,0.6); font-size: 14px; }
    .controls { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 14px; margin-top: 18px; }
    .control { display: flex; flex-direction: column; gap: 6px; font-size: 14px; }
    label { font-weight: 600; letter-spacing: 0.2px; }
    select, input[type="range"], button { border-radius: 10px; border: 1px solid rgba(255,255,255,0.1); background: rgba(255,255,255,0.04); color: #e5f0ff; padding: 8px; font-size: 14px; }
    input[type="range"] { accent-color: #41dfff; padding: 0; }
    button { cursor: pointer; transition: background 0.2s ease; }
    button:hover { background: rgba(65,223,255,0.2); }
    .hint { font-size: 13px; opacity: 0.7; margin-top: 10px; }
    .hint.small { font-size: 12px; margin-top: 4px; opacity: 0.6; }
    .inline-value { font-size: 13px; opacity: 0.75; }
    .control.full { grid-column: span 2; }
  </style>
</head>
<body>
  <div class="card">
    <h1>F767 이더넷 오실로스코프</h1>
    <p>WebSocket으로 수신한 샘플을 캔버스에 그리고, 시간축/전압축/트리거 파라미터를 즉시 조정합니다. Autoset 버튼은 최근 파형을 분석해 적절한 배율과 트리거를 추천합니다.</p>
    <div class="status">
      <div>서버 상태: <strong id="status">WebSocket 연결 대기</strong></div>
      <div id="trigger-status">트리거: -</div>
    </div>
    <div class="controls">
      <div class="control">
        <label for="time-range">시간축 <span class="inline-value" id="time-div-label">2 µs/div</span></label>
        <input type="range" id="time-range" min="0" max="0" step="1">
        <div class="hint small">0.1 µs/div부터 100 ms/div까지 연속 1-2-5 스텝</div>
      </div>
      <div class="control">
        <label for="volt-range">전압축 <span class="inline-value" id="volt-div-label">0.5 V/div</span></label>
        <input type="range" id="volt-range" min="0" max="0" step="1">
        <div class="hint small">10 mV/div ~ 10 V/div</div>
      </div>
      <div class="control">
        <label for="volt-offset">전압 오프셋 (V) <span class="inline-value" id="volt-offset-value">1.65 V</span></label>
        <input type="range" id="volt-offset" min="0" max="3.3" step="0.05" value="1.65" />
      </div>
      <div class="control">
        <label for="trigger-mode">트리거 모드</label>
        <select id="trigger-mode">
          <option value="auto">Auto</option>
          <option value="normal">Normal</option>
          <option value="single">Single</option>
        </select>
      </div>
      <div class="control">
        <label for="trigger-slope">슬로프</label>
        <select id="trigger-slope">
          <option value="rising">Rising</option>
          <option value="falling">Falling</option>
        </select>
      </div>
      <div class="control">
        <label for="trigger-channel">채널</label>
        <select id="trigger-channel">
          <option value="0">CH1</option>
          <option value="1">CH2</option>
        </select>
      </div>
      <div class="control">
        <label for="trigger-level">트리거 레벨 <span class="inline-value" id="trigger-level-value">128</span></label>
        <input type="range" id="trigger-level" min="0" max="255" value="128" />
      </div>
      <div class="control">
        <label for="trigger-holdoff">홀드오프 (µs) <span class="inline-value" id="trigger-holdoff-value">5</span></label>
        <input type="range" id="trigger-holdoff" min="0" max="100" step="1" value="5" />
      </div>
      <div class="control">
        <label>Single 모드</label>
        <button id="trigger-arm">ARM / FORCE</button>
      </div>
      <div class="control full">
        <label>Auto Set</label>
        <button id="autoset">AUTOSET</button>
      </div>
    </div>
    <canvas id="scope" width="960" height="360"></canvas>
    <p class="hint">상태 표시줄에 현재 time/div, volt/div, 트리거 상태가 표시됩니다. 모바일에서도 동일 UI가 동작합니다.</p>
  </div>
  <script>
  (function() {
    const DEFAULT_SAMPLE_RATE = 2.4e6;
    const H_DIVS = 10;
    const V_DIVS = 8;
    const FULL_SCALE_V = 3.3;
    const TIME_SCALE = build125Scale(0.1, 100000);
    const VOLT_SCALE = build125Scale(0.01, 10);

    const statusEl = document.getElementById('status');
    const triggerStatusEl = document.getElementById('trigger-status');
    const canvas = document.getElementById('scope');
    const ctx = canvas.getContext('2d');
    const controls = {
      timeRange: document.getElementById('time-range'),
      timeLabel: document.getElementById('time-div-label'),
      voltRange: document.getElementById('volt-range'),
      voltLabel: document.getElementById('volt-div-label'),
      voltOffset: document.getElementById('volt-offset'),
      voltOffsetLabel: document.getElementById('volt-offset-value'),
      mode: document.getElementById('trigger-mode'),
      slope: document.getElementById('trigger-slope'),
      channel: document.getElementById('trigger-channel'),
      level: document.getElementById('trigger-level'),
      levelLabel: document.getElementById('trigger-level-value'),
      holdoff: document.getElementById('trigger-holdoff'),
      holdoffLabel: document.getElementById('trigger-holdoff-value'),
      armBtn: document.getElementById('trigger-arm'),
      autoset: document.getElementById('autoset'),
    };
    const state = {
      timeDiv: 2,
      voltDiv: 0.5,
      voltOffset: parseFloat(controls.voltOffset.value),
      sampleRate: DEFAULT_SAMPLE_RATE,
    };
    let lastFrame = { samples: null, bits: 8, trigger: null };
    let reconnectTimer = null;
    let ws = null;

    initializeRangeControls();
    attachControlEvents();
    connect();

    function initializeRangeControls() {
      controls.timeRange.min = 0;
      controls.timeRange.max = TIME_SCALE.length - 1;
      setTimeByIndex(findNearestIndex(TIME_SCALE, state.timeDiv), true);

      controls.voltRange.min = 0;
      controls.voltRange.max = VOLT_SCALE.length - 1;
      setVoltByIndex(findNearestIndex(VOLT_SCALE, state.voltDiv), true);
    }

    function clamp(value, min, max) {
      return Math.min(Math.max(value, min), max);
    }

    function build125Scale(min, max) {
      const steps = [1, 2, 5];
      const values = [];
      let exponent = Math.floor(Math.log10(min));
      let decade = Math.pow(10, exponent);
      while (true) {
        for (const step of steps) {
          const val = Number((step * decade).toPrecision(6));
          if (val < min - 1e-9) {
            continue;
          }
          if (val > max + 1e-9) {
            if (values.length > 0) {
              return values;
            }
            continue;
          }
          values.push(val);
        }
        decade *= 10;
      }
    }

    function findNearestIndex(arr, target) {
      let best = 0;
      let diff = Infinity;
      arr.forEach((value, idx) => {
        const d = Math.abs(value - target);
        if (d < diff) {
          diff = d;
          best = idx;
        }
      });
      return best;
    }

    function formatTimeDivLabel(value) {
      const num = Number(value);
      if (!Number.isFinite(num)) return '—';
      if (num >= 1000) {
        const ms = num / 1000;
        return (Number.isInteger(ms) ? ms.toFixed(0) : ms.toFixed(2)) + ' ms/div';
      }
      if (num >= 1) {
        return (Number.isInteger(num) ? num.toFixed(0) : num.toString()) + ' µs/div';
      }
      return num.toFixed(2) + ' µs/div';
    }

    function formatVoltDivLabel(value) {
      const num = Number(value);
      if (num >= 1) {
        return num.toFixed(2) + ' V/div';
      }
      return (num * 1000).toFixed(0) + ' mV/div';
    }

    function setStatus(text) {
      statusEl.textContent = text + ' | ' + formatTimeDivLabel(state.timeDiv) + ' · ' + formatVoltDivLabel(state.voltDiv);
    }

    function setTriggerStatus(text) {
      triggerStatusEl.textContent = '트리거: ' + text;
    }

    function setTimeByIndex(index, skipRender) {
      const idx = clamp(Math.round(index), 0, TIME_SCALE.length - 1);
      state.timeDiv = TIME_SCALE[idx];
      controls.timeRange.value = idx;
      controls.timeLabel.textContent = formatTimeDivLabel(state.timeDiv);
      if (!skipRender) renderCurrentFrame();
    }

    function setVoltByIndex(index, skipRender) {
      const idx = clamp(Math.round(index), 0, VOLT_SCALE.length - 1);
      state.voltDiv = VOLT_SCALE[idx];
      controls.voltRange.value = idx;
      controls.voltLabel.textContent = formatVoltDivLabel(state.voltDiv);
      if (!skipRender) renderCurrentFrame();
    }

    function sliceForTimebase(samples, trigger) {
      const windowSeconds = state.timeDiv * 1e-6 * H_DIVS;
      let needed = Math.max(1, Math.floor(windowSeconds * state.sampleRate));
      if (!Number.isFinite(needed) || needed <= 0) {
        needed = samples.length;
      }
      if (needed > samples.length) {
        needed = samples.length;
      }
      let start = samples.length - needed;
      if (start < 0) start = 0;
      if (trigger && typeof trigger.index === 'number') {
        const trigIdx = clamp(trigger.index, 0, samples.length - 1);
        const half = Math.floor(needed / 2);
        start = trigIdx - half;
        if (start < 0) start = 0;
        if (start + needed > samples.length) {
          start = Math.max(0, samples.length - needed);
        }
      }
      const end = Math.min(samples.length, start + needed);
      return { subset: samples.slice(start, end), startIndex: start };
    }

    function renderCurrentFrame() {
      const ch = Number(controls.channel.value) || 0;
      if (!lastFrame.samples || !lastFrame.samples[ch] || !lastFrame.samples[ch].length) {
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        return;
      }
      const bits = lastFrame.bits || 8;
      const samples = lastFrame.samples[ch];
      const trig = lastFrame.trigger && lastFrame.trigger.channel === ch ? lastFrame.trigger : null;
      const { subset, startIndex } = sliceForTimebase(samples, trig);
      const maxCount = Math.pow(2, bits) - 1 || 255;
      const countsToVolt = FULL_SCALE_V / maxCount;
      let minV = state.voltOffset - (state.voltDiv * V_DIVS) / 2;
      let maxV = state.voltOffset + (state.voltDiv * V_DIVS) / 2;
      if (minV < 0) {
        maxV = clamp(maxV - minV, 0, FULL_SCALE_V);
        minV = 0;
      }
      if (maxV > FULL_SCALE_V) {
        minV = clamp(minV - (maxV - FULL_SCALE_V), 0, FULL_SCALE_V);
        maxV = FULL_SCALE_V;
      }
      const spanV = Math.max(0.01, maxV - minV);

      ctx.fillStyle = '#020611';
      ctx.fillRect(0, 0, canvas.width, canvas.height);
      ctx.strokeStyle = '#39e6ff';
      ctx.lineWidth = 2;
      ctx.beginPath();
      for (let x = 0; x < canvas.width; x++) {
        const idx = Math.min(subset.length - 1, Math.floor(x / canvas.width * subset.length));
        const volt = subset[idx] * countsToVolt;
        const norm = clamp((volt - minV) / spanV, 0, 1);
        const y = canvas.height - norm * canvas.height;
        if (x === 0) ctx.moveTo(x, y);
        else ctx.lineTo(x, y);
      }
      ctx.stroke();

      if (trig && typeof trig.level === 'number') {
        const levelVolt = (trig.level / maxCount) * FULL_SCALE_V;
        if (levelVolt >= minV && levelVolt <= maxV) {
          const levelY = canvas.height - ((levelVolt - minV) / spanV) * canvas.height;
          ctx.strokeStyle = 'rgba(255, 154, 34, 0.8)';
          ctx.setLineDash([6, 6]);
          ctx.beginPath();
          ctx.moveTo(0, levelY);
          ctx.lineTo(canvas.width, levelY);
          ctx.stroke();
          ctx.setLineDash([]);
        }
        if (typeof trig.index === 'number') {
          const rel = trig.index - startIndex;
          if (rel >= 0 && rel < subset.length) {
            const x = rel / subset.length * canvas.width;
            ctx.strokeStyle = 'rgba(255,255,255,0.6)';
            ctx.beginPath();
            ctx.moveTo(x, 0);
            ctx.lineTo(x, canvas.height);
            ctx.stroke();
          }
        }
      }
    }

    function drawWave(samples, bits, trigger) {
      if (!samples || !samples.length) {
        lastFrame = { samples: null, bits, trigger };
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        return;
      }
      lastFrame = {
        samples: samples.map((ch) => ch.slice()),
        bits,
        trigger: trigger || null,
      };
      renderCurrentFrame();
    }

    function sendTriggerConfig() {
      const payload = {
        cmd: 'set_trigger',
        mode: controls.mode.value,
        slope: controls.slope.value,
        level: Number(controls.level.value),
        holdoff_us: Number(controls.holdoff.value),
        channel: Number(controls.channel.value),
      };
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify(payload));
      }
    }

    function armSingle() {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ cmd: 'arm_single' }));
      }
    }

    function handleAutoset() {
      const ch = Number(controls.channel.value) || 0;
      if (!lastFrame.samples || !lastFrame.samples[ch] || !lastFrame.samples[ch].length) {
        return;
      }
      const samples = lastFrame.samples[ch];
      let min = samples[0];
      let max = samples[0];
      for (const v of samples) {
        if (v < min) min = v;
        if (v > max) max = v;
      }
      const mid = (min + max) / 2;
      controls.level.value = Math.round(mid);
      controls.levelLabel.textContent = controls.level.value;
      controls.mode.value = 'auto';
      controls.slope.value = 'rising';

      const bits = lastFrame.bits || 8;
      const countsToVolt = FULL_SCALE_V / (Math.pow(2, bits) - 1 || 255);
      const p2pVolt = Math.max((max - min) * countsToVolt, 0.01);
      const targetSpanPerDiv = (p2pVolt * 1.3) / V_DIVS;
      const voltIdx = findNearestIndex(VOLT_SCALE, targetSpanPerDiv);
      setVoltByIndex(voltIdx, true);

      const midVolt = clamp(mid * countsToVolt, 0, FULL_SCALE_V);
      controls.voltOffset.value = midVolt.toFixed(2);
      controls.voltOffsetLabel.textContent = midVolt.toFixed(2) + ' V';
      state.voltOffset = midVolt;

      const periodSamples = estimatePeriod(samples, mid);
      if (periodSamples) {
        const periodTime = periodSamples / state.sampleRate;
        const desiredWindow = Math.max(periodTime * 2, periodTime * 1.2);
        const desiredPerDiv = (desiredWindow / H_DIVS) * 1e6;
        const timeIdx = findNearestIndex(TIME_SCALE, desiredPerDiv);
        setTimeByIndex(timeIdx, true);
      }

      renderCurrentFrame();
      sendTriggerConfig();
    }

    function estimatePeriod(samples, threshold) {
      let first = -1;
      for (let i = 1; i < samples.length; i++) {
        if (samples[i - 1] < threshold && samples[i] >= threshold) {
          if (first === -1) {
            first = i;
          } else {
            return i - first;
          }
        }
      }
      return null;
    }

    function attachControlEvents() {
      controls.timeRange.addEventListener('input', () => {
        setTimeByIndex(Number(controls.timeRange.value), false);
      });
      controls.voltRange.addEventListener('input', () => {
        setVoltByIndex(Number(controls.voltRange.value), false);
      });
      controls.voltOffset.addEventListener('input', () => {
        state.voltOffset = parseFloat(controls.voltOffset.value);
        controls.voltOffsetLabel.textContent = state.voltOffset.toFixed(2) + ' V';
        renderCurrentFrame();
      });
      controls.mode.addEventListener('change', sendTriggerConfig);
      controls.slope.addEventListener('change', sendTriggerConfig);
      controls.channel.addEventListener('change', () => {
        sendTriggerConfig();
        renderCurrentFrame();
      });
      controls.level.addEventListener('input', () => {
        controls.levelLabel.textContent = controls.level.value;
        sendTriggerConfig();
      });
      controls.holdoff.addEventListener('input', () => {
        controls.holdoffLabel.textContent = controls.holdoff.value;
        sendTriggerConfig();
      });
      controls.armBtn.addEventListener('click', (e) => {
        e.preventDefault();
        armSingle();
      });
      controls.autoset.addEventListener('click', (e) => {
        e.preventDefault();
        handleAutoset();
      });
    }

    function connect() {
      const url = (window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host + '/ws';
      ws = new WebSocket(url);

      ws.onopen = () => {
        setStatus('WebSocket 연결됨 · 샘플 대기 중');
        if (reconnectTimer) {
          clearTimeout(reconnectTimer);
          reconnectTimer = null;
        }
        sendTriggerConfig();
      };

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.samples && msg.samples.length) {
            if (typeof msg.sample_rate === 'number' && isFinite(msg.sample_rate) && msg.sample_rate > 0) {
              state.sampleRate = msg.sample_rate;
            } else {
              state.sampleRate = DEFAULT_SAMPLE_RATE;
            }
            drawWave(msg.samples, msg.sample_bits || 8, msg.trigger);
            const trigState = msg.trigger && msg.trigger.state ? msg.trigger.state : 'auto';
            const info = trigState + ' · ' + msg.samples_per_ch + ' samples';
            setStatus('seq ' + msg.seq + ' · ' + info);
            const trigMode = msg.trigger && msg.trigger.mode ? msg.trigger.mode.toUpperCase() : 'AUTO';
            setTriggerStatus(trigState + ' @ ' + trigMode);
          }
        } catch (err) {
          console.error('invalid ws message', err);
        }
      };

      ws.onclose = () => {
        setStatus('연결 끊김 · 재연결 시도 중…');
        setTriggerStatus('재연결 중');
        reconnectTimer = setTimeout(connect, 1500);
      };

      ws.onerror = () => {
        ws.close();
      };
    }
  })();
  </script>

</body>
</html>`

const (
	protoHeaderSize  = 0x14
	defaultPreview   = 8
	approxSampleRate = 2.4e6 // samples per second per channel
)

type packetHeader struct {
	PacketSeq      uint32
	FirstSampleIdx uint64
	Channels       uint16
	SamplesPerCh   uint16
	Flags          uint16
	SampleBits     uint16
}

type triggerMode string

const (
	triggerModeAuto   triggerMode = "auto"
	triggerModeNormal triggerMode = "normal"
	triggerModeSingle triggerMode = "single"
)

type triggerSlope string

const (
	triggerSlopeRising  triggerSlope = "rising"
	triggerSlopeFalling triggerSlope = "falling"
)

type triggerConfig struct {
	Mode      triggerMode
	Slope     triggerSlope
	Level     uint8   // 0-255 slider range
	HoldoffUs float64 // microseconds
	Channel   int
}

type triggerInfo struct {
	Mode      string  `json:"mode"`
	Slope     string  `json:"slope"`
	Level     uint16  `json:"level"`
	HoldoffUs float64 `json:"holdoff_us"`
	Channel   int     `json:"channel"`
	State     string  `json:"state"`
	Active    bool    `json:"active"`
	Index     int     `json:"index"`
}

type triggerController struct {
	mu             sync.RWMutex
	cfg            triggerConfig
	lastTriggerIdx uint64
	singleArmed    bool
	sampleRate     float64
}

type triggerUpdate struct {
	Mode      string  `json:"mode"`
	Slope     string  `json:"slope"`
	Level     float64 `json:"level"`
	HoldoffUs float64 `json:"holdoff_us"`
	Channel   int     `json:"channel"`
}

type wsCommand struct {
	Cmd string `json:"cmd"`

	Mode      string  `json:"mode,omitempty"`
	Slope     string  `json:"slope,omitempty"`
	Level     float64 `json:"level,omitempty"`
	HoldoffUs float64 `json:"holdoff_us,omitempty"`
	Channel   int     `json:"channel,omitempty"`
}

type packetEvent struct {
	Seq            uint32      `json:"seq"`
	FirstSampleIdx uint64      `json:"first_idx"`
	Channels       uint16      `json:"channels"`
	SamplesPerCh   uint16      `json:"samples_per_ch"`
	SampleBits     uint16      `json:"sample_bits"`
	Flags          uint16      `json:"flags"`
	Samples        [][]uint16  `json:"samples"`
	Trigger        triggerInfo `json:"trigger"`
}

func newTriggerController() *triggerController {
	return &triggerController{
		cfg: triggerConfig{
			Mode:      triggerModeAuto,
			Slope:     triggerSlopeRising,
			Level:     128,
			HoldoffUs: 5,
			Channel:   0,
		},
		singleArmed: true,
		sampleRate:  approxSampleRate,
	}
}

func (tc *triggerController) Config() triggerConfig {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.cfg
}

func (tc *triggerController) Update(upd triggerUpdate) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if upd.Mode != "" {
		switch triggerMode(upd.Mode) {
		case triggerModeAuto, triggerModeNormal, triggerModeSingle:
			tc.cfg.Mode = triggerMode(upd.Mode)
			if tc.cfg.Mode != triggerModeSingle {
				tc.singleArmed = true
			} else {
				tc.singleArmed = false
			}
		}
	}
	if upd.Slope != "" {
		switch triggerSlope(upd.Slope) {
		case triggerSlopeRising, triggerSlopeFalling:
			tc.cfg.Slope = triggerSlope(upd.Slope)
		}
	}
	if upd.Level >= 0 {
		if upd.Level > 255 {
			upd.Level = 255
		}
		tc.cfg.Level = uint8(upd.Level)
	}
	if upd.HoldoffUs >= 0 {
		tc.cfg.HoldoffUs = upd.HoldoffUs
	}
	if upd.Channel >= 0 {
		tc.cfg.Channel = upd.Channel
	}
}

func (tc *triggerController) ArmSingle() {
	tc.mu.Lock()
	tc.singleArmed = true
	tc.mu.Unlock()
}

func (tc *triggerController) Process(h packetHeader, payload []byte) (bool, triggerInfo, error) {
	tc.mu.RLock()
	cfg := tc.cfg
	lastIdx := tc.lastTriggerIdx
	singleArmed := tc.singleArmed
	tc.mu.RUnlock()

	infos := triggerInfo{
		Mode:      string(cfg.Mode),
		Slope:     string(cfg.Slope),
		HoldoffUs: cfg.HoldoffUs,
		Channel:   cfg.Channel,
		Index:     -1,
	}

	if cfg.Mode == triggerModeSingle && !singleArmed {
		infos.State = "latched"
		return false, infos, nil
	}

	sampleBytes := int(h.SampleBits / 8)
	if sampleBytes <= 0 || sampleBytes > 2 {
		return false, infos, fmt.Errorf("sample bits %d not supported for trigger", h.SampleBits)
	}

	channel := cfg.Channel
	if channel < 0 || channel >= int(h.Channels) {
		channel = 0
	}
	infos.Channel = channel

	orig := extractOriginalSamples(payload, h, channel)
	if len(orig) == 0 {
		return false, infos, errors.New("empty sample payload")
	}

	maxValue := (1 << h.SampleBits) - 1
	level := uint16((int(cfg.Level) * maxValue) / 255)
	infos.Level = level

	sampleCount := int(h.SamplesPerCh)
	var prev uint16
	found := -1

	for i := 0; i < sampleCount; i++ {
		start := i * sampleBytes
		val := uint16(decodeSample(orig[start : start+sampleBytes]))
		if i > 0 {
			switch cfg.Slope {
			case triggerSlopeRising:
				if prev < level && val >= level {
					found = i
				}
			case triggerSlopeFalling:
				if prev > level && val <= level {
					found = i
				}
			}
			if found != -1 {
				break
			}
		}
		prev = val
	}

	holdoffSamples := uint64(cfg.HoldoffUs * tc.sampleRate / 1e6)
	shouldSend := true

	if found == -1 {
		switch cfg.Mode {
		case triggerModeAuto:
			infos.State = "auto"
			shouldSend = true
		case triggerModeNormal:
			infos.State = "waiting"
			shouldSend = false
		case triggerModeSingle:
			if singleArmed {
				infos.State = "armed"
			} else {
				infos.State = "latched"
			}
			shouldSend = false
		}
		return shouldSend, infos, nil
	}

	absoluteIdx := h.FirstSampleIdx + uint64(found)
	if holdoffSamples > 0 && absoluteIdx-lastIdx < holdoffSamples {
		infos.State = "holdoff"
		switch cfg.Mode {
		case triggerModeAuto:
			shouldSend = true
		case triggerModeNormal, triggerModeSingle:
			shouldSend = false
		}
		return shouldSend, infos, nil
	}

	infos.Active = true
	infos.Index = found
	infos.State = "triggered"

	tc.mu.Lock()
	tc.lastTriggerIdx = absoluteIdx
	if cfg.Mode == triggerModeSingle {
		if singleArmed {
			tc.singleArmed = false
		} else {
			shouldSend = false
		}
	}
	tc.mu.Unlock()

	return shouldSend, infos, nil
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
	trigger       *triggerController
}

func newWSHub(fps int, trigger *triggerController) *wsHub {
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
		trigger:       trigger,
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

	triggerCtl := newTriggerController()

	hub := newWSHub(*uiFPS, triggerCtl)
	hub.Start()

	go func() {
		if err := runUDPReceiver(*udpListen, *dumpPackets, hub, triggerCtl); err != nil {
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

func runUDPReceiver(listenAddr string, dumpPackets bool, hub *wsHub, trigger *triggerController) error {
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

		shouldSend := true
		trigInfo := triggerInfo{
			Mode:  string(triggerModeAuto),
			Slope: string(triggerSlopeRising),
			Index: -1,
			State: "passthrough",
		}
		if trigger != nil {
			var trigErr error
			shouldSend, trigInfo, trigErr = trigger.Process(header, payload)
			if trigErr != nil {
				trigInfo.State = "error"
				log.Printf("trigger processing error seq=%d: %v", header.PacketSeq, trigErr)
			}
			if !shouldSend {
				continue
			}
		}

		msg, err := buildPacketEvent(header, payload, trigInfo)
		if err != nil {
			log.Printf("event marshal error seq=%d: %v", header.PacketSeq, err)
			continue
		}
		hub.EnqueuePacket(header.PacketSeq, msg)
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

func extractOriginalSamples(payload []byte, h packetHeader, channel int) []byte {
	sampleBytes := int(h.SampleBits / 8)
	if sampleBytes <= 0 {
		return nil
	}
	perChannel := (int(h.SamplesPerCh) * sampleBytes * 2) + sampleBytes
	if perChannel <= 0 {
		return nil
	}
	if channel < 0 {
		channel = 0
	}
	if channel >= int(h.Channels) {
		channel = int(h.Channels) - 1
	}
	offset := channel * perChannel
	start := offset
	end := offset + int(h.SamplesPerCh)*sampleBytes
	if start < 0 || end > len(payload) || start >= end {
		return nil
	}
	return payload[start:end]
}

func buildPacketEvent(h packetHeader, payload []byte, trig triggerInfo) ([]byte, error) {
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
		Trigger:        trig,
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

func (h *wsHub) EnqueuePacket(seq uint32, data []byte) {
	if len(data) == 0 {
		return
	}
	h.latest.Store(seq, data)
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
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("ws read error: %v", err)
			}
			break
		}
		if msgType == websocket.TextMessage {
			h.handleCommand(data)
		}
	}
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

func (h *wsHub) handleCommand(data []byte) {
	if h.trigger == nil {
		return
	}

	var cmd wsCommand
	if err := json.Unmarshal(data, &cmd); err != nil {
		log.Printf("ws command decode error: %v", err)
		return
	}

	switch cmd.Cmd {
	case "set_trigger":
		update := triggerUpdate{
			Mode:      cmd.Mode,
			Slope:     cmd.Slope,
			Level:     cmd.Level,
			HoldoffUs: cmd.HoldoffUs,
			Channel:   cmd.Channel,
		}
		h.trigger.Update(update)
	case "arm_single":
		h.trigger.ArmSingle()
	default:
		log.Printf("ws unknown command: %s", cmd.Cmd)
	}
}
