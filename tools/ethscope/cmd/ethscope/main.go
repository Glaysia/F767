package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
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
    .tab-buttons { display: flex; gap: 10px; margin-bottom: 14px; }
    .tab-btn { border-radius: 10px; border: 1px solid rgba(255,255,255,0.1); background: rgba(255,255,255,0.06); color: #e5f0ff; padding: 10px 14px; cursor: pointer; font-weight: 700; letter-spacing: 0.3px; transition: background 0.15s ease, border-color 0.15s ease; }
    .tab-btn.active { background: linear-gradient(120deg, rgba(65,223,255,0.3), rgba(255,255,255,0.08)); border-color: rgba(65,223,255,0.6); }
    .tab-pane { display: none; }
    .tab-pane.active { display: block; }
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
    .panel-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); gap: 14px; margin-top: 18px; }
    .panel { background: rgba(255,255,255,0.03); border: 1px solid rgba(73,123,177,0.35); border-radius: 14px; padding: 14px 16px; }
    .panel-header { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 10px; }
    .panel-title { font-weight: 700; letter-spacing: 0.3px; }
    .panel-sub { font-size: 12px; opacity: 0.7; margin-top: 2px; }
    .panel-actions { display: flex; align-items: center; gap: 8px; font-size: 13px; }
    .panel-actions label { font-weight: 600; }
    .metric-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(120px, 1fr)); gap: 10px; }
    .metric { background: rgba(0,0,0,0.15); border: 1px solid rgba(255,255,255,0.06); border-radius: 10px; padding: 10px 12px; }
    .metric label { display: block; font-size: 12px; opacity: 0.75; }
    .metric .value { font-size: 18px; font-weight: 700; margin-top: 4px; color: #41dfff; letter-spacing: 0.2px; }
    .cursor-section { display: flex; flex-direction: column; gap: 8px; }
    .cursor-row { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 10px; font-size: 13px; }
    .cursor-sliders { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 10px; }
    .cursor-sliders label { font-size: 12px; opacity: 0.85; display: flex; flex-direction: column; gap: 4px; }
    .readout { font-family: "JetBrains Mono", "SFMono-Regular", monospace; font-size: 12px; opacity: 0.85; }
    .fg-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 14px; margin-top: 12px; }
    .fg-card { background: rgba(255,255,255,0.03); border: 1px solid rgba(73,123,177,0.35); border-radius: 14px; padding: 14px 16px; }
    .fg-raw { width: 100%; padding: 10px 12px; border-radius: 10px; border: 1px solid rgba(255,255,255,0.1); background: rgba(0,0,0,0.25); color: #e5f0ff; font-size: 14px; }
    .fg-status { margin-top: 12px; padding: 12px 14px; border-radius: 12px; border: 1px dashed rgba(90,140,200,0.6); background: rgba(255,255,255,0.03); font-size: 13px; }
    .button-row { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 8px; }
    .raw-row { display: flex; gap: 10px; align-items: center; margin-top: 6px; }
  </style>
</head>
<body>
  <div class="card">
    <div class="tab-buttons">
      <button class="tab-btn active" data-tab="scope">오실로스코프</button>
      <button class="tab-btn" data-tab="fg">함수 발생기</button>
    </div>

    <div id="tab-scope" class="tab-pane active">
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
          <label for="volt-offset">전압 오프셋 (V) <span class="inline-value" id="volt-offset-value">0.00 V</span></label>
          <input type="range" id="volt-offset" min="-9" max="9" step="0.05" value="0" />
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
          <label for="trigger-level">트리거 레벨 (V) <span class="inline-value" id="trigger-level-value">0.00 V</span></label>
          <input type="range" id="trigger-level" min="-9" max="9" step="0.05" value="0" />
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
      <div class="panel-grid">
        <div class="panel">
          <div class="panel-header">
            <div>
              <div class="panel-title">메져</div>
              <div class="panel-sub">Vpp · Mean · RMS · Freq · T</div>
            </div>
            <div class="panel-actions">
              <label for="measure-channel">채널</label>
              <select id="measure-channel">
                <option value="0">CH1</option>
                <option value="1">CH2</option>
              </select>
            </div>
          </div>
          <div class="metric-grid">
            <div class="metric">
              <label>Vpp</label>
              <div class="value" id="measure-vpp">-</div>
            </div>
            <div class="metric">
              <label>Mean</label>
              <div class="value" id="measure-mean">-</div>
            </div>
            <div class="metric">
              <label>RMS</label>
              <div class="value" id="measure-rms">-</div>
            </div>
            <div class="metric">
              <label>Freq</label>
              <div class="value" id="measure-freq">-</div>
            </div>
            <div class="metric">
              <label>T (주기)</label>
              <div class="value" id="measure-period">-</div>
            </div>
          </div>
        </div>
        <div class="panel">
          <div class="panel-header">
            <div>
              <div class="panel-title">커서</div>
              <div class="panel-sub">시간/전압 커서 Δt, ΔV 표시</div>
            </div>
          </div>
          <div class="cursor-section">
            <div class="cursor-row">
              <label><input type="checkbox" id="cursor-time-enable" checked> 시간 커서</label>
              <div class="readout" id="cursor-time-readout">T1=- · T2=- · Δt=-</div>
            </div>
            <div class="cursor-sliders">
              <label>T1 위치 (%)<input type="range" id="cursor-t1" min="0" max="100" step="1" value="30"></label>
              <label>T2 위치 (%)<input type="range" id="cursor-t2" min="0" max="100" step="1" value="70"></label>
            </div>
            <div class="cursor-row">
              <label><input type="checkbox" id="cursor-volt-enable"> 전압 커서</label>
              <div class="readout" id="cursor-volt-readout">V1=- · V2=- · ΔV=-</div>
            </div>
            <div class="cursor-sliders">
              <label>V1 (V)<input type="range" id="cursor-v1" min="-9" max="9" step="0.05" value="1.0"></label>
              <label>V2 (V)<input type="range" id="cursor-v2" min="-9" max="9" step="0.05" value="-1.0"></label>
            </div>
          </div>
        </div>
      </div>
      <canvas id="scope" width="960" height="360"></canvas>
      <p class="hint">상태 표시줄에 현재 time/div, volt/div, 트리거 상태가 표시됩니다. 모바일에서도 동일 UI가 동작합니다.</p>
    </div>

    <div id="tab-fg" class="tab-pane">
      <h1>함수 발생기 컨트롤</h1>
      <p>PC→MCU1(이더넷)→UART4→MCU2(FG) 릴레이 경로를 사용합니다. UART 115200/8/N/1, 명령은 UART.md 표준을 그대로 보냅니다.</p>
      <div class="fg-grid">
        <div class="fg-card">
          <div class="control">
            <label for="fg-waveform">파형 선택</label>
            <select id="fg-waveform">
              <option value="0">SINE (0)</option>
              <option value="1">SQUARE (1)</option>
              <option value="2">TRIANGLE (2)</option>
              <option value="3">SAWTOOTH (3)</option>
            </select>
            <div class="button-row">
              <button type="button" data-fg-wave="0">SINE</button>
              <button type="button" data-fg-wave="1">SQUARE</button>
              <button type="button" data-fg-wave="2">TRIANGLE</button>
              <button type="button" data-fg-wave="3">SAWTOOTH</button>
            </div>
          </div>
        </div>
        <div class="fg-card">
          <div class="control">
            <label for="fg-freq">주파수 (Hz) <span class="inline-value" id="fg-freq-value">1000</span></label>
            <input type="range" id="fg-freq" min="100" max="100000" step="50" value="1000">
            <div class="hint small">100~100000 Hz (정수)</div>
            <div class="button-row">
              <button type="button" id="fg-apply-freq">주파수 적용</button>
            </div>
          </div>
        </div>
        <div class="fg-card">
          <div class="control">
            <label for="fg-amp">진폭 (DAC 코드) <span class="inline-value" id="fg-amp-value">4095</span></label>
            <input type="range" id="fg-amp" min="0" max="4095" step="1" value="4095">
            <div class="hint small">0~4095 (full-scale=4095)</div>
            <div class="button-row">
              <button type="button" id="fg-apply-amp">진폭 적용</button>
            </div>
          </div>
        </div>
        <div class="fg-card">
          <div class="control">
            <label for="fg-raw">직접 명령 (예: F1000, A2048, D, H)</label>
            <div class="raw-row">
              <input id="fg-raw" class="fg-raw" type="text" placeholder="W0 / F1000 / A4095 / D / H">
              <button type="button" id="fg-send-raw">전송</button>
            </div>
            <div class="button-row">
              <button type="button" id="fg-display">D (현재 설정)</button>
              <button type="button" id="fg-help">H (Help)</button>
            </div>
          </div>
        </div>
      </div>
      <div class="fg-status" id="fg-status">대기: WebSocket 연결 후 명령을 보낼 수 있습니다.</div>
      <p class="hint small">UART4 TX=PC10, RX=PC11 (115200/8/N/1). 명령은 자동으로 LF로 끝납니다.</p>
    </div>
  </div>
  <script>
  (function() {
    const DEFAULT_SAMPLE_RATE = 1.263157e6; // match TIM5 (timer=72 MHz, ARR=56 -> 72e6/(56+1))
    const H_DIVS = 10;
    const V_DIVS = 8;
    const FULL_SCALE_MIN_V = -9;
    const FULL_SCALE_MAX_V = 9;
    const FULL_SCALE_SPAN_V = FULL_SCALE_MAX_V - FULL_SCALE_MIN_V;
    const RING_CAPACITY = 500000;
    const MAX_DISPLAY_POINTS = 2048;
    const CHANNEL_COLORS = ['#ffd447', '#4fb7ff', '#8df5ff', '#ff7ceb'];
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
    const measureUI = {
      channel: document.getElementById('measure-channel'),
      vpp: document.getElementById('measure-vpp'),
      mean: document.getElementById('measure-mean'),
      rms: document.getElementById('measure-rms'),
      freq: document.getElementById('measure-freq'),
      period: document.getElementById('measure-period'),
    };
    const cursorUI = {
      timeEnable: document.getElementById('cursor-time-enable'),
      voltEnable: document.getElementById('cursor-volt-enable'),
      t1: document.getElementById('cursor-t1'),
      t2: document.getElementById('cursor-t2'),
      v1: document.getElementById('cursor-v1'),
      v2: document.getElementById('cursor-v2'),
      timeReadout: document.getElementById('cursor-time-readout'),
      voltReadout: document.getElementById('cursor-volt-readout'),
    };
    const tabButtons = Array.from(document.querySelectorAll('.tab-btn'));
    const tabPanes = {
      scope: document.getElementById('tab-scope'),
      fg: document.getElementById('tab-fg'),
    };
    const fgUI = {
      waveform: document.getElementById('fg-waveform'),
      freq: document.getElementById('fg-freq'),
      freqValue: document.getElementById('fg-freq-value'),
      amp: document.getElementById('fg-amp'),
      ampValue: document.getElementById('fg-amp-value'),
      raw: document.getElementById('fg-raw'),
      sendRaw: document.getElementById('fg-send-raw'),
      applyFreq: document.getElementById('fg-apply-freq'),
      applyAmp: document.getElementById('fg-apply-amp'),
      displayBtn: document.getElementById('fg-display'),
      helpBtn: document.getElementById('fg-help'),
      status: document.getElementById('fg-status'),
      quickWaveButtons: document.querySelectorAll('[data-fg-wave]'),
    };
    const state = {
      timeDiv: 2,
      voltDiv: 0.5,
      voltOffset: parseFloat(controls.voltOffset.value),
      sampleRate: DEFAULT_SAMPLE_RATE,
      sampleBits: 8,
      triggerLevelVolt: parseFloat(document.getElementById('trigger-level').value),
    };
    const cursorState = {
      timeEnabled: true,
      voltEnabled: false,
      t1: parseInt(cursorUI.t1.value, 10) / 100,
      t2: parseInt(cursorUI.t2.value, 10) / 100,
      v1: parseFloat(cursorUI.v1.value),
      v2: parseFloat(cursorUI.v2.value),
    };
    if (!isFinite(cursorState.t1)) cursorState.t1 = 0.3;
    if (!isFinite(cursorState.t2)) cursorState.t2 = 0.7;
    if (!isFinite(cursorState.v1)) cursorState.v1 = 1;
    if (!isFinite(cursorState.v2)) cursorState.v2 = -1;
    let currentView = {
      windowSeconds: 0,
      sampleRate: state.sampleRate,
      channels: [],
      minV: 0,
      maxV: 0,
    };
    const ring = {
      buffers: [],
      capacity: RING_CAPACITY,
    };
    let lastMsg = null;
    let lastTriggerInfo = null;
    let lastTriggerAbsIdx = null;
    let reconnectTimer = null;
    let ws = null;

    function clearTriggerAnchor() {
      lastTriggerAbsIdx = null;
    }

    initializeRangeControls();
    attachControlEvents();
    attachTabEvents();
    attachFgEvents();
    connect();

    function initializeRangeControls() {
      controls.timeRange.min = 0;
      controls.timeRange.max = TIME_SCALE.length - 1;
      setTimeByIndex(findNearestIndex(TIME_SCALE, state.timeDiv), true);

      controls.voltRange.min = 0;
      controls.voltRange.max = VOLT_SCALE.length - 1;
      setVoltByIndex(findNearestIndex(VOLT_SCALE, state.voltDiv), true);
      controls.levelLabel.textContent = formatVolt(state.triggerLevelVolt);
    }

    function attachTabEvents() {
      tabButtons.forEach((btn) => {
        btn.addEventListener('click', () => {
          const target = btn.dataset.tab || 'scope';
          switchTab(target);
        });
      });
    }

    function switchTab(target) {
      Object.keys(tabPanes).forEach((key) => {
        const active = key === target;
        if (tabPanes[key]) {
          tabPanes[key].classList.toggle('active', active);
        }
      });
      tabButtons.forEach((btn) => {
        btn.classList.toggle('active', btn.dataset.tab === target);
      });
    }

    function setFgStatus(text, isError) {
      if (!fgUI.status) return;
      fgUI.status.textContent = text;
      fgUI.status.style.color = isError ? '#ffb3b3' : '#e5f0ff';
    }

    function sendFgCommand(line, label) {
      const trimmed = (line || '').trim();
      if (!trimmed) {
        setFgStatus('명령이 비었습니다.', true);
        return;
      }
      if (!ws || ws.readyState !== WebSocket.OPEN) {
        setFgStatus('WebSocket 미연결: ' + trimmed, true);
        return;
      }
      ws.send(JSON.stringify({ cmd: 'fg_send', line: trimmed }));
      setFgStatus((label || '보냄') + ': ' + trimmed);
    }

    function updateFgLabels() {
      if (fgUI.freq && fgUI.freqValue) {
        const hz = Math.round(parseFloat(fgUI.freq.value) || 0);
        fgUI.freqValue.textContent = hz;
      }
      if (fgUI.amp && fgUI.ampValue) {
        const amp = Math.round(parseFloat(fgUI.amp.value) || 0);
        fgUI.ampValue.textContent = amp;
      }
    }

    function attachFgEvents() {
      if (fgUI.waveform) {
        fgUI.waveform.addEventListener('change', () => {
          sendFgCommand('W' + fgUI.waveform.value, '파형');
        });
      }
      fgUI.quickWaveButtons.forEach((btn) => {
        btn.addEventListener('click', () => {
          const code = btn.dataset.fgWave;
          sendFgCommand(code, '파형');
        });
      });
      if (fgUI.freq) {
        fgUI.freq.addEventListener('input', updateFgLabels);
      }
      if (fgUI.applyFreq) {
        fgUI.applyFreq.addEventListener('click', () => {
          const hz = clamp(Math.round(parseFloat(fgUI.freq.value) || 0), 100, 100000);
          fgUI.freq.value = hz;
          updateFgLabels();
          sendFgCommand('F' + hz, '주파수');
        });
      }
      if (fgUI.amp) {
        fgUI.amp.addEventListener('input', updateFgLabels);
      }
      if (fgUI.applyAmp) {
        fgUI.applyAmp.addEventListener('click', () => {
          const amp = clamp(Math.round(parseFloat(fgUI.amp.value) || 0), 0, 4095);
          fgUI.amp.value = amp;
          updateFgLabels();
          sendFgCommand('A' + amp, '진폭');
        });
      }
      if (fgUI.sendRaw && fgUI.raw) {
        fgUI.sendRaw.addEventListener('click', () => {
          sendFgCommand(fgUI.raw.value, '직접');
        });
        fgUI.raw.addEventListener('keydown', (e) => {
          if (e.key === 'Enter') {
            e.preventDefault();
            sendFgCommand(fgUI.raw.value, '직접');
          }
        });
      }
      if (fgUI.displayBtn) {
        fgUI.displayBtn.addEventListener('click', () => sendFgCommand('D', '상태 요청'));
      }
      if (fgUI.helpBtn) {
        fgUI.helpBtn.addEventListener('click', () => sendFgCommand('H', 'Help'));
      }
      updateFgLabels();
    }

    function clamp(value, min, max) {
      return Math.min(Math.max(value, min), max);
    }

    function build125Scale(min, max) {
      const steps = [1, 2, 5];
      const values = [];
      let decade = Math.pow(10, Math.floor(Math.log10(min)));
      while (values.length === 0 || values[values.length - 1] < max) {
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
      return values;
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
      if (value >= 1000) {
        const ms = value / 1000;
        return (Number.isInteger(ms) ? ms.toFixed(0) : ms.toFixed(2)) + ' ms/div';
      }
      if (value >= 1) {
        return (Number.isInteger(value) ? value.toFixed(0) : value.toFixed(2)) + ' µs/div';
      }
      return value.toFixed(2) + ' µs/div';
    }

    function formatVoltDivLabel(value) {
      if (value >= 1) {
        return value.toFixed(2) + ' V/div';
      }
      return (value * 1000).toFixed(0) + ' mV/div';
    }

    function formatVoltTick(value) {
      const abs = Math.abs(value);
      if (abs >= 10) return value.toFixed(0) + ' V';
      if (abs >= 1) return value.toFixed(1) + ' V';
      if (abs >= 0.1) return value.toFixed(2) + ' V';
      return (value * 1000).toFixed(0) + ' mV';
    }

    function formatVolt(value) {
      return value.toFixed(2) + ' V';
    }

    function formatTimeTick(microseconds) {
      const abs = Math.abs(microseconds);
      if (abs >= 1e6) {
        const seconds = microseconds / 1e6;
        return seconds.toFixed(abs >= 1e7 ? 0 : 2) + ' s';
      }
      if (abs >= 1000) {
        const ms = microseconds / 1000;
        const precision = ms >= 100 ? 0 : ms >= 10 ? 1 : 2;
        return ms.toFixed(precision) + ' ms';
      }
      const precision = abs >= 100 ? 0 : abs >= 10 ? 1 : 2;
      return microseconds.toFixed(precision) + ' µs';
    }

    function formatVoltValue(value) {
      if (!isFinite(value)) return '-';
      const abs = Math.abs(value);
      if (abs >= 1) return value.toFixed(2) + ' V';
      return (value * 1000).toFixed(1) + ' mV';
    }

    function formatFreqHz(freq) {
      if (!isFinite(freq) || freq <= 0) return '-';
      if (freq >= 1e6) return (freq / 1e6).toFixed(freq >= 10e6 ? 1 : 2) + ' MHz';
      if (freq >= 1e3) return (freq / 1e3).toFixed(freq >= 100e3 ? 1 : 2) + ' kHz';
      return freq.toFixed(freq >= 100 ? 0 : 2) + ' Hz';
    }

    function formatSeconds(sec) {
      if (!isFinite(sec) || sec <= 0) return '-';
      return formatTimeTick(sec * 1e6);
    }

    function formatDeltaTime(sec) {
      if (!isFinite(sec)) return '-';
      const sign = sec >= 0 ? '' : '-';
      return sign + formatTimeTick(Math.abs(sec) * 1e6);
    }

    function formatDeltaVolt(volt) {
      if (!isFinite(volt)) return '-';
      const abs = Math.abs(volt);
      if (abs >= 1) return volt.toFixed(2) + ' V';
      return (volt * 1000).toFixed(1) + ' mV';
    }

    function setStatus(text) {
      statusEl.textContent = text + ' | ' + formatTimeDivLabel(state.timeDiv) + ' · ' + formatVoltDivLabel(state.voltDiv);
    }

    function setTriggerStatus(text) {
      triggerStatusEl.textContent = '트리거: ' + text;
    }

    function voltsToCounts(volts, bits) {
      const maxCount = Math.max(1, (1 << bits) - 1);
      const countsToVolt = FULL_SCALE_SPAN_V / maxCount;
      const clampedVolt = clamp(volts, FULL_SCALE_MIN_V, FULL_SCALE_MAX_V);
      const counts = Math.round((clampedVolt - FULL_SCALE_MIN_V) / countsToVolt);
      return clamp(counts, 0, maxCount);
    }

    function countsToVolts(counts, bits) {
      const maxCount = Math.max(1, (1 << bits) - 1);
      const countsToVolt = FULL_SCALE_SPAN_V / maxCount;
      return FULL_SCALE_MIN_V + counts * countsToVolt;
    }

    function findTriggerIndex(samples, levelCounts, slope) {
      if (!samples || samples.length < 2) return -1;
      let prev = samples[0];
      for (let i = 1; i < samples.length; i++) {
        const val = samples[i];
        if (slope === 'rising') {
          if (prev < levelCounts && val >= levelCounts) return i;
        } else if (slope === 'falling') {
          if (prev > levelCounts && val <= levelCounts) return i;
        }
        prev = val;
      }
      return -1;
    }

    function setTimeByIndex(index, skipRender) {
      const idx = clamp(Math.round(index), 0, TIME_SCALE.length - 1);
      state.timeDiv = TIME_SCALE[idx];
      controls.timeRange.value = idx;
      controls.timeLabel.textContent = formatTimeDivLabel(state.timeDiv);
      requestViewSpan();
      if (!skipRender) renderCurrentFrame();
    }

    function requestViewSpan() {
      if (!ws || ws.readyState !== WebSocket.OPEN) {
        return;
      }
      const windowSeconds = state.timeDiv * 1e-6 * H_DIVS;
      const samples = Math.max(1, Math.floor(windowSeconds * state.sampleRate));
      ws.send(JSON.stringify({ cmd: 'set_view', samples }));
    }

    function setVoltByIndex(index, skipRender) {
      const idx = clamp(Math.round(index), 0, VOLT_SCALE.length - 1);
      state.voltDiv = VOLT_SCALE[idx];
      controls.voltRange.value = idx;
      controls.voltLabel.textContent = formatVoltDivLabel(state.voltDiv);
      if (!skipRender) renderCurrentFrame();
    }

    function ensureBuffers(count, startIdx) {
      while (ring.buffers.length < count) {
        ring.buffers.push({
          data: new Uint16Array(ring.capacity),
          head: 0,
          size: 0,
          startIdx: startIdx || 0,
          endIdx: startIdx || 0,
        });
      }
    }

    function resetBuffer(buf, startIdx) {
      buf.head = 0;
      buf.size = 0;
      buf.startIdx = startIdx;
      buf.endIdx = startIdx;
    }

    function pushSample(buf, absIdx, value) {
      if (buf.size === 0) {
        buf.startIdx = absIdx;
        buf.endIdx = absIdx;
      }
      if (absIdx !== buf.endIdx) {
        resetBuffer(buf, absIdx);
      }
      buf.data[buf.head] = value;
      buf.head = (buf.head + 1) % ring.capacity;
      if (buf.size < ring.capacity) {
        buf.size++;
      } else {
        buf.startIdx++;
      }
      buf.endIdx = buf.startIdx + buf.size;
    }

    function appendSamples(msg) {
      ensureBuffers(msg.samples.length, msg.first_idx);
      msg.samples.forEach((channelSamples, ch) => {
        const buf = ring.buffers[ch];
        const msgStart = msg.first_idx;
        const msgEnd = msg.first_idx + channelSamples.length;
        if (buf.size === 0 || msgEnd <= buf.startIdx) {
          resetBuffer(buf, msgStart);
        }
        const appendFrom = Math.max(msgStart, buf.endIdx);
        const startOffset = Math.max(0, appendFrom - msg.first_idx);
        for (let i = startOffset; i < channelSamples.length; i++) {
          const absIdx = msg.first_idx + i;
          pushSample(buf, absIdx, channelSamples[i]);
        }
      });
    }

    function ringSnapshot(buf, sampleCount) {
      if (!buf || buf.size === 0) {
        return { data: [], startIdx: buf ? buf.endIdx : 0 };
      }
      let count = Math.min(sampleCount, buf.size);
      const result = new Array(count);
      let idx = (buf.head - count + ring.capacity) % ring.capacity;
      for (let i = 0; i < count; i++) {
        result[i] = buf.data[idx];
        idx = (idx + 1) % ring.capacity;
      }
      const startIdx = buf.endIdx - count;
      return { data: result, startIdx };
    }

    function ringRange(buf, startIdx, sampleCount) {
      if (!buf || buf.size === 0 || sampleCount <= 0) {
        return { data: [], startIdx: startIdx || 0 };
      }
      const firstAvail = buf.startIdx;
      const lastAvail = buf.endIdx;
      let start = startIdx;
      if (start < firstAvail) start = firstAvail;
      if (start > lastAvail - 1) start = lastAvail - 1;
      if (start + sampleCount > lastAvail) {
        start = lastAvail - sampleCount;
      }
      if (start < firstAvail) {
        sampleCount = Math.max(0, sampleCount - (firstAvail - start));
        start = firstAvail;
      }
      if (sampleCount <= 0) {
        return { data: [], startIdx: start };
      }
      const result = new Array(sampleCount);
      let offset = (start - buf.startIdx) % ring.capacity;
      for (let i = 0; i < sampleCount; i++) {
        result[i] = buf.data[(offset + i) % ring.capacity];
      }
      return { data: result, startIdx: start };
    }

    function downsample(data, maxPoints) {
      if (!data.length) {
        return { mins: [], maxs: [] };
      }
      if (data.length <= maxPoints) {
        return { mins: data.slice(), maxs: data.slice() };
      }
      const mins = new Array(maxPoints);
      const maxs = new Array(maxPoints);
      const ratio = data.length / maxPoints;
      for (let i = 0; i < maxPoints; i++) {
        const start = Math.floor(i * ratio);
        let end = Math.floor((i + 1) * ratio);
        if (end <= start) end = start + 1;
        if (end > data.length) end = data.length;
        let minVal = data[start];
        let maxVal = data[start];
        for (let j = start + 1; j < end; j++) {
          const val = data[j];
          if (val < minVal) minVal = val;
          if (val > maxVal) maxVal = val;
        }
        mins[i] = minVal;
        maxs[i] = maxVal;
      }
      return { mins, maxs };
    }

    function drawAxes() {
      ctx.fillStyle = '#020611';
      ctx.fillRect(0, 0, canvas.width, canvas.height);
      ctx.strokeStyle = 'rgba(255,255,255,0.75)';
      ctx.lineWidth = 1;
      ctx.setLineDash([4, 6]);
      for (let i = 1; i < H_DIVS; i++) {
        const x = (canvas.width / H_DIVS) * i;
        ctx.beginPath();
        ctx.moveTo(x, 0);
        ctx.lineTo(x, canvas.height);
        ctx.stroke();
      }
      for (let i = 1; i < V_DIVS; i++) {
        const y = (canvas.height / V_DIVS) * i;
        ctx.beginPath();
        ctx.moveTo(0, y);
        ctx.lineTo(canvas.width, y);
        ctx.stroke();
      }
      ctx.setLineDash([]);
      // Axis lines
      ctx.strokeStyle = 'rgba(255,255,255,0.35)';
      ctx.lineWidth = 1;
      const midX = canvas.width / 2;
      const midY = canvas.height / 2;
      ctx.beginPath();
      ctx.moveTo(midX, 0);
      ctx.lineTo(midX, canvas.height);
      ctx.stroke();
      ctx.beginPath();
      ctx.moveTo(0, midY);
      ctx.lineTo(canvas.width, midY);
      ctx.stroke();
    }

    function renderCurrentFrame() {
      drawAxes();
      const bits = Math.max(1, state.sampleBits || 8);
      const maxCount = (1 << bits) - 1;
      const countsToVolt = FULL_SCALE_SPAN_V / maxCount;
      const windowSeconds = state.timeDiv * 1e-6 * H_DIVS;
      const neededSamples = Math.max(1, Math.floor(windowSeconds * state.sampleRate));
      const minV = state.voltOffset - (state.voltDiv * V_DIVS) / 2;
      const maxV = state.voltOffset + (state.voltDiv * V_DIVS) / 2;
      const spanV = Math.max(0.01, maxV - minV);
      const trigChannel = lastTriggerInfo ? lastTriggerInfo.channel : null;
      const trigLevelCounts = lastTriggerInfo ? lastTriggerInfo.level : null;

      currentView = {
        windowSeconds,
        sampleRate: state.sampleRate,
        channels: [],
        minV,
        maxV,
      };
      let windowStartIdx = null;

      ring.buffers.forEach((buf, ch) => {
        if (buf.size === 0) {
          return;
        }

        if (windowStartIdx === null) {
          const latestStart = buf.endIdx - neededSamples;
          const triggerIdx = typeof lastTriggerAbsIdx === 'number' ? lastTriggerAbsIdx : null;
          if (triggerIdx !== null) {
            windowStartIdx = triggerIdx - Math.floor(neededSamples / 2);
          } else {
            windowStartIdx = latestStart;
          }
        }

        const snapshot = ringRange(buf, windowStartIdx, neededSamples);
        if (snapshot.data.length) {
          currentView.channels[ch] = {
            samples: snapshot.data,
            startIdx: snapshot.startIdx,
            sampleRate: state.sampleRate,
          };
        }
        if (!snapshot.data.length) {
          return;
        }
        const ds = downsample(snapshot.data, MAX_DISPLAY_POINTS);
        const points = ds.mins.length;
        if (points) {
          const color = CHANNEL_COLORS[ch % CHANNEL_COLORS.length];
          ctx.save();
          ctx.strokeStyle = color;
          ctx.globalAlpha = 0.35;
          ctx.lineWidth = 1;
          for (let idx = 0; idx < points; idx++) {
            const x = (idx / (points - 1 || 1)) * canvas.width;
            const voltMin = FULL_SCALE_MIN_V + ds.mins[idx] * countsToVolt;
            const voltMax = FULL_SCALE_MIN_V + ds.maxs[idx] * countsToVolt;
            const normMin = clamp((voltMin - minV) / spanV, 0, 1);
            const normMax = clamp((voltMax - minV) / spanV, 0, 1);
            const yMin = canvas.height - normMin * canvas.height;
            const yMax = canvas.height - normMax * canvas.height;
            ctx.beginPath();
            ctx.moveTo(x, yMax);
            ctx.lineTo(x, yMin);
            ctx.stroke();
          }
          ctx.restore();

          ctx.strokeStyle = color;
          ctx.lineWidth = 2;
          ctx.beginPath();
          for (let idx = 0; idx < points; idx++) {
            const x = (idx / (points - 1 || 1)) * canvas.width;
            const mid = (ds.mins[idx] + ds.maxs[idx]) / 2;
            const volt = FULL_SCALE_MIN_V + mid * countsToVolt;
            const norm = clamp((volt - minV) / spanV, 0, 1);
            const y = canvas.height - norm * canvas.height;
            if (idx === 0) ctx.moveTo(x, y);
            else ctx.lineTo(x, y);
          }
          ctx.stroke();
        }

        if (trigChannel === ch && typeof trigLevelCounts === 'number') {
          const trigVolt = FULL_SCALE_MIN_V + trigLevelCounts * countsToVolt;
          if (trigVolt >= minV && trigVolt <= maxV) {
            const norm = clamp((trigVolt - minV) / spanV, 0, 1);
            const y = canvas.height - norm * canvas.height;
            ctx.strokeStyle = 'rgba(255,255,255,0.35)';
            ctx.setLineDash([6, 4]);
            ctx.beginPath();
            ctx.moveTo(0, y);
            ctx.lineTo(canvas.width, y);
            ctx.stroke();
            ctx.setLineDash([]);
          }
        }

        if (lastTriggerInfo && lastTriggerInfo.channel === ch && typeof lastTriggerAbsIdx === 'number') {
          const rel = lastTriggerAbsIdx - snapshot.startIdx;
          if (rel >= 0 && rel < snapshot.data.length) {
            const frac = rel / (snapshot.data.length - 1 || 1);
            const x = frac * canvas.width;
            ctx.strokeStyle = 'rgba(255,255,255,0.6)';
            ctx.setLineDash([4, 4]);
            ctx.beginPath();
            ctx.moveTo(x, 0);
            ctx.lineTo(x, canvas.height);
            ctx.stroke();
            ctx.setLineDash([]);
          }
        }
      });

      drawCursors(minV, maxV, windowSeconds);
      drawGridLabels(minV, maxV);
      drawScaleLabels();
      updateMeasurements();
      updateCursorReadouts(windowSeconds);
    }

    function drawGridLabels(minV, maxV) {
      const labelColor = CHANNEL_COLORS[0] || '#ffd447';
      ctx.save();
      ctx.fillStyle = labelColor;
      ctx.font = '11px "Segoe UI", "Pretendard", sans-serif';
      ctx.textAlign = 'left';
      ctx.textBaseline = 'middle';
      const padding = 6;
      for (let i = 0; i <= V_DIVS; i++) {
        const y = (canvas.height / V_DIVS) * i;
        const v = maxV - i * state.voltDiv;
        ctx.fillText(formatVoltTick(v), padding, y);
      }

      ctx.textAlign = 'center';
      ctx.textBaseline = 'bottom';
      const bottomPadding = 6;
      const halfDivs = H_DIVS / 2;
      for (let i = 0; i <= H_DIVS; i++) {
        const x = (canvas.width / H_DIVS) * i;
        const tUs = (i - halfDivs) * state.timeDiv;
        ctx.fillText(formatTimeTick(tUs), x, canvas.height - bottomPadding);
      }
      ctx.restore();
    }

    function drawScaleLabels() {
      ctx.save();
      ctx.fillStyle = 'rgba(255,255,255,0.7)';
      ctx.font = '12px "Segoe UI", "Pretendard", sans-serif';
      ctx.textBaseline = 'bottom';
      const padding = 10;
      const timeText = formatTimeDivLabel(state.timeDiv);
      const voltText = formatVoltDivLabel(state.voltDiv);
      ctx.fillText(timeText, padding, canvas.height - padding - 14);
      ctx.fillText(voltText, padding, canvas.height - padding);
      ctx.restore();
    }

    function drawCursors(minV, maxV, windowSeconds) {
      const spanV = Math.max(0.001, maxV - minV);
      ctx.save();
      if (cursorState.timeEnabled) {
        const tColor = 'rgba(255,255,255,0.65)';
        const positions = [
          { frac: clamp(cursorState.t1, 0, 1), label: 'T1' },
          { frac: clamp(cursorState.t2, 0, 1), label: 'T2' },
        ];
        ctx.strokeStyle = tColor;
        ctx.setLineDash([4, 4]);
        positions.forEach((pos) => {
          const x = pos.frac * canvas.width;
          ctx.beginPath();
          ctx.moveTo(x, 0);
          ctx.lineTo(x, canvas.height);
          ctx.stroke();
          ctx.fillStyle = tColor;
          ctx.font = '12px "Segoe UI", "Pretendard", sans-serif';
          ctx.textBaseline = 'top';
          ctx.fillText(pos.label, clamp(x + 4, 0, canvas.width - 20), 6);
        });
        ctx.setLineDash([]);
      }

      if (cursorState.voltEnabled) {
        const vColor = '#41dfff';
        const vPositions = [
          { value: cursorState.v1, label: 'V1' },
          { value: cursorState.v2, label: 'V2' },
        ];
        ctx.strokeStyle = vColor;
        ctx.setLineDash([6, 4]);
        vPositions.forEach((p) => {
          const clampedV = clamp(p.value, minV, maxV);
          const norm = clamp((clampedV - minV) / spanV, 0, 1);
          const y = canvas.height - norm * canvas.height;
          ctx.beginPath();
          ctx.moveTo(0, y);
          ctx.lineTo(canvas.width, y);
          ctx.stroke();
          ctx.fillStyle = vColor;
          ctx.font = '12px "Segoe UI", "Pretendard", sans-serif';
          ctx.textBaseline = 'bottom';
          ctx.fillText(p.label, 6, clamp(y - 4, 12, canvas.height - 4));
        });
        ctx.setLineDash([]);
      }
      ctx.restore();
    }

    function estimateFrequency(samples, threshold, sampleRate) {
      if (!samples || samples.length < 2 || !sampleRate) return null;
      const edges = [];
      for (let i = 1; i < samples.length; i++) {
        if (samples[i - 1] < threshold && samples[i] >= threshold) {
          edges.push(i);
          if (edges.length >= 8) break;
        }
      }
      if (edges.length < 2) return null;
      let total = 0;
      for (let i = 1; i < edges.length; i++) {
        total += edges[i] - edges[i - 1];
      }
      const avg = total / (edges.length - 1);
      if (!isFinite(avg) || avg <= 0) return null;
      return sampleRate / avg;
    }

    function updateMeasurements() {
      const ch = Number(measureUI.channel.value) || 0;
      const view = currentView.channels[ch];
      if (!view || !view.samples || !view.samples.length) {
        measureUI.vpp.textContent = '-';
        measureUI.mean.textContent = '-';
        measureUI.rms.textContent = '-';
        measureUI.freq.textContent = '-';
        measureUI.period.textContent = '-';
        return;
      }
      const bits = Math.max(1, state.sampleBits || 8);
      const maxCount = (1 << bits) - 1;
      const countsToVolt = FULL_SCALE_SPAN_V / maxCount;
      let minC = view.samples[0];
      let maxC = view.samples[0];
      let sumVolt = 0;
      let sumSqVolt = 0;
      for (const c of view.samples) {
        if (c < minC) minC = c;
        if (c > maxC) maxC = c;
        const v = FULL_SCALE_MIN_V + c * countsToVolt;
        sumVolt += v;
        sumSqVolt += v * v;
      }
      const meanVolt = sumVolt / view.samples.length;
      const rmsVolt = Math.sqrt(sumSqVolt / view.samples.length);
      const vpp = (maxC - minC) * countsToVolt;
      const level = (minC + maxC) / 2;
      const freq = estimateFrequency(view.samples, level, view.sampleRate || currentView.sampleRate);
      const period = freq ? 1 / freq : null;

      measureUI.vpp.textContent = formatVoltValue(vpp);
      measureUI.mean.textContent = formatVoltValue(meanVolt);
      measureUI.rms.textContent = formatVoltValue(rmsVolt);
      measureUI.freq.textContent = formatFreqHz(freq);
      measureUI.period.textContent = formatSeconds(period);
    }

    function updateCursorReadouts(windowSeconds) {
      const spanSec = windowSeconds || currentView.windowSeconds;
      const t1 = clamp(cursorState.t1, 0, 1);
      const t2 = clamp(cursorState.t2, 0, 1);
      if (cursorState.timeEnabled && spanSec > 0) {
        const t1Sec = (t1 - 0.5) * spanSec;
        const t2Sec = (t2 - 0.5) * spanSec;
        const deltaSec = (t2 - t1) * spanSec;
        const freq = deltaSec !== 0 ? Math.abs(1 / deltaSec) : null;
        cursorUI.timeReadout.textContent =
          'T1=' + formatDeltaTime(t1Sec) +
          ' · T2=' + formatDeltaTime(t2Sec) +
          ' · Δt=' + formatDeltaTime(deltaSec) +
          (freq ? ' · 1/Δt=' + formatFreqHz(freq) : '');
      } else if (cursorState.timeEnabled) {
        cursorUI.timeReadout.textContent = '시간 커서 대기 (화면 없음)';
      } else {
        cursorUI.timeReadout.textContent = '시간 커서 OFF';
      }

      if (cursorState.voltEnabled) {
        const deltaV = cursorState.v2 - cursorState.v1;
        cursorUI.voltReadout.textContent =
          'V1=' + formatVoltValue(cursorState.v1) +
          ' · V2=' + formatVoltValue(cursorState.v2) +
          ' · ΔV=' + formatDeltaVolt(deltaV);
      } else {
        cursorUI.voltReadout.textContent = '전압 커서 OFF';
      }
    }

    function sendTriggerConfig() {
      state.triggerLevelVolt = parseFloat(controls.level.value);
      const bits = Math.max(1, state.sampleBits || 8);
      const levelCounts = voltsToCounts(state.triggerLevelVolt, bits);
      const payload = {
        cmd: 'set_trigger',
        mode: controls.mode.value,
        slope: controls.slope.value,
        level: levelCounts,
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
      if (!lastMsg || !lastMsg.samples || !lastMsg.samples.length) {
        return;
      }
      const ch = Number(controls.channel.value) || 0;
      const samples = lastMsg.samples[ch];
      if (!samples || !samples.length) return;
      let min = samples[0];
      let max = samples[0];
      for (const v of samples) {
        if (v < min) min = v;
        if (v > max) max = v;
      }
      const bits = lastMsg.sample_bits || 8;
      const countsToVolt = FULL_SCALE_SPAN_V / ((1 << bits) - 1 || 255);
      const mid = (min + max) / 2;
      const midVolt = clamp(FULL_SCALE_MIN_V + mid * countsToVolt, FULL_SCALE_MIN_V, FULL_SCALE_MAX_V);
      controls.level.value = midVolt.toFixed(2);
      controls.levelLabel.textContent = formatVolt(midVolt);
      controls.mode.value = 'auto';
      controls.slope.value = 'rising';
      const p2pVolt = Math.max((max - min) * countsToVolt, 0.01);
      const targetSpanPerDiv = (p2pVolt * 1.3) / V_DIVS;
      setVoltByIndex(findNearestIndex(VOLT_SCALE, targetSpanPerDiv), true);
      controls.voltOffset.value = midVolt.toFixed(2);
      controls.voltOffsetLabel.textContent = midVolt.toFixed(2) + ' V';
      state.voltOffset = midVolt;
      const periodSamples = estimatePeriod(samples, mid);
      if (periodSamples) {
        const periodTime = periodSamples / state.sampleRate;
        const desiredWindow = Math.max(periodTime * 2, periodTime * 1.2);
        const desiredPerDiv = (desiredWindow / H_DIVS) * 1e6;
        setTimeByIndex(findNearestIndex(TIME_SCALE, desiredPerDiv), true);
      }
      renderCurrentFrame();
      sendTriggerConfig();
    }

    function estimatePeriod(samples, threshold) {
      let first = -1;
      for (let i = 1; i < samples.length; i++) {
        if (samples[i - 1] < threshold && samples[i] >= threshold) {
          if (first === -1) first = i;
          else return i - first;
        }
      }
      return null;
    }

    function attachControlEvents() {
      controls.timeRange.addEventListener('input', () => {
        setTimeByIndex(Number(controls.timeRange.value));
      });
      controls.voltRange.addEventListener('input', () => {
        setVoltByIndex(Number(controls.voltRange.value));
      });
      controls.voltOffset.addEventListener('input', () => {
        state.voltOffset = parseFloat(controls.voltOffset.value);
        controls.voltOffsetLabel.textContent = state.voltOffset.toFixed(2) + ' V';
        renderCurrentFrame();
      });
      controls.mode.addEventListener('change', () => {
        clearTriggerAnchor();
        sendTriggerConfig();
        renderCurrentFrame();
      });
      controls.slope.addEventListener('change', sendTriggerConfig);
      controls.channel.addEventListener('change', () => {
        sendTriggerConfig();
        renderCurrentFrame();
      });
      controls.level.addEventListener('input', () => {
        state.triggerLevelVolt = parseFloat(controls.level.value);
        controls.levelLabel.textContent = formatVolt(state.triggerLevelVolt);
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
      measureUI.channel.addEventListener('change', () => {
        updateMeasurements();
      });
      cursorUI.timeEnable.addEventListener('change', () => {
        cursorState.timeEnabled = cursorUI.timeEnable.checked;
        renderCurrentFrame();
      });
      cursorUI.voltEnable.addEventListener('change', () => {
        cursorState.voltEnabled = cursorUI.voltEnable.checked;
        renderCurrentFrame();
      });
      cursorUI.t1.addEventListener('input', () => {
        cursorState.t1 = parseInt(cursorUI.t1.value, 10) / 100;
        renderCurrentFrame();
      });
      cursorUI.t2.addEventListener('input', () => {
        cursorState.t2 = parseInt(cursorUI.t2.value, 10) / 100;
        renderCurrentFrame();
      });
      cursorUI.v1.addEventListener('input', () => {
        cursorState.v1 = parseFloat(cursorUI.v1.value);
        renderCurrentFrame();
      });
      cursorUI.v2.addEventListener('input', () => {
        cursorState.v2 = parseFloat(cursorUI.v2.value);
        renderCurrentFrame();
      });
    }

    function connect() {
      const url = (window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host + '/ws';
      ws = new WebSocket(url);
      ws.onopen = () => {
        setStatus('WebSocket 연결됨 · 샘플 대기 중');
        setFgStatus('연결됨: FG 명령 전송 가능');
        if (reconnectTimer) {
          clearTimeout(reconnectTimer);
          reconnectTimer = null;
        }
        sendTriggerConfig();
        requestViewSpan();
      };
      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.samples && msg.samples.length) {
            appendSamples(msg);
            lastMsg = msg;
            state.sampleRate = msg.sample_rate || DEFAULT_SAMPLE_RATE;
            state.sampleBits = msg.sample_bits || 8;
            lastTriggerInfo = msg.trigger || null;
            const triggerActive =
              msg.trigger &&
              typeof msg.trigger.index === 'number' &&
              msg.trigger.index >= 0 &&
              (msg.trigger.state === 'triggered' || msg.trigger.active);
            if (triggerActive) {
              if (typeof msg.trigger_abs_idx === 'number') {
                lastTriggerAbsIdx = msg.trigger_abs_idx;
              } else {
                lastTriggerAbsIdx = msg.first_idx + msg.trigger.index;
              }
            } else {
              // Try client-side edge detection on the latest packet to anchor slow waveforms.
              const ch = Number(controls.channel.value) || 0;
              const bits = state.sampleBits || 8;
              const levelCounts = voltsToCounts(state.triggerLevelVolt, bits);
              if (msg.samples[ch]) {
                const idx = findTriggerIndex(msg.samples[ch], levelCounts, controls.slope.value);
                if (idx >= 0) {
                  lastTriggerAbsIdx = msg.first_idx + idx;
                  lastTriggerInfo = {
                    channel: ch,
                    level: levelCounts,
                    mode: controls.mode.value,
                    slope: controls.slope.value,
                    state: 'client-detected',
                    active: true,
                    index: idx,
                  };
                }
              }
            }
            renderCurrentFrame();
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
        setFgStatus('연결 끊김 · 명령 대기', true);
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
	approxSampleRate = 1.263157e6 // samples per second per channel (TIM5 timer clk 72MHz, ARR=56)
	snapshotSamples  = 16384
	displayPoints    = 2048
	eventSchemaVer   = 1
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
	Samples   int     `json:"samples,omitempty"`
	Line      string  `json:"line,omitempty"`
}

type packetEvent struct {
	Seq            uint32      `json:"seq"`
	FirstSampleIdx uint64      `json:"first_idx"`
	SampleRate     float64     `json:"sample_rate"`
	Channels       uint16      `json:"channels"`
	SamplesPerCh   int         `json:"samples_per_ch"`
	SampleBits     uint16      `json:"sample_bits"`
	Flags          uint16      `json:"flags"`
	Samples        [][]uint16  `json:"samples"`
	SamplesMin     [][]uint16  `json:"samples_min,omitempty"`
	SamplesMax     [][]uint16  `json:"samples_max,omitempty"`
	HistorySeconds float64     `json:"history_seconds"`
	BufferUtil     float64     `json:"buffer_utilization"`
	DropCount      uint64      `json:"drop_count"`
	IngestDelayUs  uint64      `json:"ingest_delay_us"`
	TriggerAbsIdx  uint64      `json:"trigger_abs_idx"`
	SchemaVersion  int         `json:"schema_version"`
	Trigger        triggerInfo `json:"trigger"`
}

type captureJob struct {
	header   packetHeader
	samples  [][]uint16
	received time.Time
}

type channelRing struct {
	data []uint16
	head int
	size int
}

func newChannelRing(capacity int) *channelRing {
	return &channelRing{
		data: make([]uint16, capacity),
		head: 0,
		size: 0,
	}
}

func (r *channelRing) append(samples []uint16) int {
	if len(samples) == 0 || len(r.data) == 0 {
		return 0
	}
	dropped := 0
	for _, v := range samples {
		if r.size == len(r.data) {
			dropped++
		} else {
			r.size++
		}
		r.data[r.head] = v
		r.head = (r.head + 1) % len(r.data)
	}
	return dropped
}

func (r *channelRing) snapshot(maxSamples int) []uint16 {
	if r.size == 0 {
		return nil
	}
	if maxSamples > r.size {
		maxSamples = r.size
	}
	out := make([]uint16, maxSamples)
	start := (r.head - maxSamples + len(r.data)) % len(r.data)
	for i := 0; i < maxSamples; i++ {
		out[i] = r.data[(start+i)%len(r.data)]
	}
	return out
}

type sampleBuffer struct {
	mu          sync.RWMutex
	rings       []*channelRing
	startIdx    uint64
	expectedIdx uint64

	sampleBits uint16
	channels   uint16
	lastSeq    uint32
	lastFlags  uint16

	lastTrigger      triggerInfo
	lastTriggerAbs   uint64
	version          atomic.Uint64
	sampleRate       float64
	samplesPerPacket uint16
	ringCapacity     int
	historySeconds   float64
	dropCount        uint64
	ingestLagUs      uint64
}

func newSampleBuffer(channels int, capacity int, historySeconds float64, sampleRate float64) *sampleBuffer {
	if channels <= 0 {
		channels = 1
	}
	if capacity <= 0 {
		capacity = snapshotSamples
	}
	rings := make([]*channelRing, channels)
	for i := range rings {
		rings[i] = newChannelRing(capacity)
	}
	return &sampleBuffer{
		rings:          rings,
		startIdx:       0,
		sampleBits:     8,
		channels:       uint16(channels),
		sampleRate:     sampleRate,
		ringCapacity:   capacity,
		historySeconds: historySeconds,
	}
}

func (sb *sampleBuffer) reset(startIdx uint64, channels int) {
	if channels <= 0 {
		channels = 1
	}
	capacity := sb.ringCapacity
	if capacity <= 0 && len(sb.rings) > 0 && len(sb.rings[0].data) > 0 {
		capacity = len(sb.rings[0].data)
	}
	rings := make([]*channelRing, channels)
	for i := range rings {
		rings[i] = newChannelRing(capacity)
	}
	sb.rings = rings
	sb.startIdx = startIdx
	sb.expectedIdx = startIdx
	sb.channels = uint16(channels)
	sb.version.Add(1)
}

func (sb *sampleBuffer) appendPacket(h packetHeader, samples [][]uint16, trig triggerInfo, ingestDelay time.Duration, sampleRate float64) error {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	if len(samples) == 0 {
		return errors.New("no samples to append")
	}

	if len(sb.rings) != len(samples) {
		sb.reset(h.FirstSampleIdx, len(samples))
	}

	if sb.expectedIdx == 0 && sb.startIdx == 0 {
		sb.startIdx = h.FirstSampleIdx
		sb.expectedIdx = h.FirstSampleIdx
	}

	if h.FirstSampleIdx != sb.expectedIdx {
		if h.FirstSampleIdx > sb.expectedIdx {
			gap := h.FirstSampleIdx - sb.expectedIdx
			sb.dropCount += gap
			log.Printf("capture gap: seq=%d expected_idx=%d got_idx=%d gap=%d drops=%d", h.PacketSeq, sb.expectedIdx, h.FirstSampleIdx, gap, sb.dropCount)
		} else {
			log.Printf("capture rewind: seq=%d expected_idx=%d got_idx=%d (resetting buffer)", h.PacketSeq, sb.expectedIdx, h.FirstSampleIdx)
		}
		if h.FirstSampleIdx > sb.expectedIdx {
			sb.dropCount += h.FirstSampleIdx - sb.expectedIdx
		}
		sb.reset(h.FirstSampleIdx, len(samples))
	}

	var dropped int
	for ch, r := range sb.rings {
		if len(samples[ch]) == 0 {
			continue
		}
		if ch == 0 {
			dropped = r.append(samples[ch])
		} else {
			r.append(samples[ch])
		}
	}

	if dropped > 0 {
		sb.startIdx += uint64(dropped)
		sb.dropCount += uint64(dropped)
		if len(sb.rings) > 0 && len(sb.rings[0].data) > 0 {
			log.Printf("ring overflow: seq=%d first_idx=%d dropped=%d cap=%d", h.PacketSeq, h.FirstSampleIdx, dropped, len(sb.rings[0].data))
		} else {
			log.Printf("ring overflow: seq=%d first_idx=%d dropped=%d", h.PacketSeq, h.FirstSampleIdx, dropped)
		}
	}
	sb.expectedIdx = h.FirstSampleIdx + uint64(len(samples[0]))
	sb.sampleBits = h.SampleBits
	sb.channels = h.Channels
	sb.lastSeq = h.PacketSeq
	sb.lastFlags = h.Flags
	sb.samplesPerPacket = h.SamplesPerCh
	if sampleRate > 0 {
		sb.sampleRate = sampleRate
	} else {
		sb.sampleRate = approxSampleRate
	}
	sb.lastTrigger = trig
	sb.lastTriggerAbs = 0
	if trig.Active && trig.Index >= 0 {
		sb.lastTriggerAbs = h.FirstSampleIdx + uint64(trig.Index)
	}
	if ingestDelay < 0 {
		ingestDelay = 0
	}
	sb.ingestLagUs = uint64(ingestDelay / time.Microsecond)
	sb.version.Add(1)
	return nil
}

func (sb *sampleBuffer) snapshot(maxSamples int) (packetEvent, uint64, bool) {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	if len(sb.rings) == 0 || sb.rings[0].size == 0 {
		return packetEvent{}, 0, false
	}

	evt := packetEvent{
		Seq:            sb.lastSeq,
		SampleRate:     sb.sampleRate,
		Channels:       uint16(len(sb.rings)),
		SampleBits:     sb.sampleBits,
		Flags:          sb.lastFlags,
		Samples:        make([][]uint16, len(sb.rings)),
		SamplesMin:     make([][]uint16, len(sb.rings)),
		SamplesMax:     make([][]uint16, len(sb.rings)),
		Trigger:        sb.lastTrigger,
		SamplesPerCh:   0,
		HistorySeconds: sb.historySeconds,
		DropCount:      sb.dropCount,
		IngestDelayUs:  sb.ingestLagUs,
		TriggerAbsIdx:  sb.lastTriggerAbs,
		SchemaVersion:  eventSchemaVer,
	}

	firstIdx := sb.startIdx
	if len(sb.rings) > 0 && len(sb.rings[0].data) > 0 {
		evt.BufferUtil = float64(sb.rings[0].size) / float64(len(sb.rings[0].data))
	}
	for ch, r := range sb.rings {
		chSamples := r.snapshot(maxSamples)
		evt.Samples[ch] = chSamples
		if len(chSamples) == 0 {
			continue
		}
		if evt.SamplesPerCh == 0 || len(chSamples) < evt.SamplesPerCh {
			evt.SamplesPerCh = len(chSamples)
		}
	}

	if evt.SamplesPerCh == 0 {
		return packetEvent{}, 0, false
	}

	// Align all channels to the shortest snapshot to keep time bases consistent.
	for ch, s := range evt.Samples {
		if len(s) > evt.SamplesPerCh {
			evt.Samples[ch] = s[len(s)-evt.SamplesPerCh:]
		}
	}

	if len(sb.rings) > 0 {
		currentSize := sb.rings[0].size
		if currentSize < evt.SamplesPerCh {
			evt.FirstSampleIdx = firstIdx
		} else {
			offset := currentSize - evt.SamplesPerCh
			evt.FirstSampleIdx = firstIdx + uint64(offset)
		}
	} else {
		evt.FirstSampleIdx = firstIdx
	}
	if evt.TriggerAbsIdx >= evt.FirstSampleIdx {
		rel := evt.TriggerAbsIdx - evt.FirstSampleIdx
		if rel < uint64(evt.SamplesPerCh) {
			evt.Trigger.Index = int(rel)
		} else {
			evt.Trigger.Index = -1
		}
	} else {
		evt.Trigger.Index = -1
	}

	for ch, data := range evt.Samples {
		if len(data) == 0 {
			continue
		}
		mins, maxs := minMaxDownsample(data, displayPoints)
		evt.SamplesMin[ch] = mins
		evt.SamplesMax[ch] = maxs
	}

	return evt, sb.version.Load(), true
}

func minMaxDownsample(samples []uint16, maxPoints int) ([]uint16, []uint16) {
	if maxPoints <= 0 || len(samples) == 0 {
		return nil, nil
	}
	if len(samples) <= maxPoints {
		mins := append([]uint16(nil), samples...)
		maxs := append([]uint16(nil), samples...)
		return mins, maxs
	}

	mins := make([]uint16, maxPoints)
	maxs := make([]uint16, maxPoints)
	ratio := float64(len(samples)) / float64(maxPoints)
	for i := 0; i < maxPoints; i++ {
		start := int(math.Floor(float64(i) * ratio))
		end := int(math.Floor(float64(i+1) * ratio))
		if end <= start {
			end = start + 1
		}
		if end > len(samples) {
			end = len(samples)
		}
		minVal := samples[start]
		maxVal := samples[start]
		for j := start + 1; j < end; j++ {
			v := samples[j]
			if v < minVal {
				minVal = v
			}
			if v > maxVal {
				maxVal = v
			}
		}
		mins[i] = minVal
		maxs[i] = maxVal
	}
	return mins, maxs
}

func findTriggerIndex(samples []uint16, level uint16, slope triggerSlope) int {
	if len(samples) == 0 {
		return -1
	}
	prev := samples[0]
	for i := 1; i < len(samples); i++ {
		val := samples[i]
		switch slope {
		case triggerSlopeRising:
			if prev < level && val >= level {
				return i
			}
		case triggerSlopeFalling:
			if prev > level && val <= level {
				return i
			}
		}
		prev = val
	}
	return -1
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

func (tc *triggerController) SetSampleRate(rate float64) {
	if rate <= 0 {
		return
	}
	tc.mu.Lock()
	tc.sampleRate = rate
	tc.mu.Unlock()
}

func (tc *triggerController) ArmSingle() {
	tc.mu.Lock()
	tc.singleArmed = true
	tc.mu.Unlock()
}

func (tc *triggerController) Process(h packetHeader, samples [][]uint16) (bool, triggerInfo, error) {
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

	channel := cfg.Channel
	if channel < 0 || channel >= int(h.Channels) {
		channel = 0
	}
	infos.Channel = channel

	if len(samples) == 0 || len(samples[0]) == 0 {
		return false, infos, errors.New("empty sample payload")
	}
	if channel >= len(samples) {
		channel = 0
	}
	channelSamples := samples[channel]

	if h.SampleBits == 0 {
		return false, infos, errors.New("invalid sample bits")
	}
	maxValue := (1 << h.SampleBits) - 1
	level := uint16((int(cfg.Level) * maxValue) / 255)
	infos.Level = level

	sampleCount := len(channelSamples)
	var prev uint16
	found := -1

	for i := 0; i < sampleCount; i++ {
		val := channelSamples[i]
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

type fgRelay struct {
	mu   sync.Mutex
	conn *net.UDPConn
	addr *net.UDPAddr
}

func newFGRelay(dest string) (*fgRelay, error) {
	if strings.TrimSpace(dest) == "" {
		return nil, nil
	}
	addr, err := net.ResolveUDPAddr("udp", dest)
	if err != nil {
		return nil, fmt.Errorf("resolve fg addr: %w", err)
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("dial fg addr: %w", err)
	}
	return &fgRelay{
		conn: conn,
		addr: addr,
	}, nil
}

func (f *fgRelay) Send(line string) error {
	if f == nil || f.conn == nil || f.addr == nil {
		return errors.New("fg relay not configured")
	}
	text := strings.TrimSpace(line)
	if text == "" {
		return errors.New("empty fg command")
	}
	// Function generator expects newline-terminated ASCII commands.
	payload := []byte(text)
	if !strings.HasSuffix(text, "\n") && !strings.HasSuffix(text, "\r") {
		payload = append(payload, '\n')
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	_, err := f.conn.Write(payload)
	return err
}

type wsHub struct {
	mu            sync.Mutex
	clients       map[*websocket.Conn]struct{}
	upgrader      websocket.Upgrader
	frameInterval time.Duration
	writeTimeout  time.Duration
	lastBroadcast atomic.Uint64
	trigger       *triggerController
	buffer        *sampleBuffer
	snapshotSize  int
	sampleRate    float64
	fg            *fgRelay
}

func newWSHub(fps int, trigger *triggerController, historySamples int, historySeconds float64, sampleRate float64, fg *fgRelay) *wsHub {
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
		buffer:        newSampleBuffer(1, historySamples, historySeconds, sampleRate),
		snapshotSize:  snapshotSamples,
		sampleRate:    sampleRate,
		fg:            fg,
	}
}

func (h *wsHub) updateSnapshotSize(samples int) {
	if samples <= 0 {
		return
	}
	if h.buffer == nil {
		return
	}
	maxSamples := h.buffer.ringCapacity
	if samples > maxSamples {
		samples = maxSamples
	}
	const minSamples = snapshotSamples
	if samples < minSamples {
		samples = minSamples
	}
	h.snapshotSize = samples
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

func (h *wsHub) appendPacket(hdr packetHeader, samples [][]uint16, trig triggerInfo, ingestDelay time.Duration) {
	if h.buffer == nil {
		return
	}
	if err := h.buffer.appendPacket(hdr, samples, trig, ingestDelay, h.sampleRate); err != nil {
		log.Printf("buffer append failed seq=%d: %v", hdr.PacketSeq, err)
	}
}

func main() {
	listen := flag.String("listen", ":8080", "HTTP listen address, e.g. :8080 or 0.0.0.0:8080")
	udpListen := flag.String("udp", ":5000", "UDP listen address for ADC packets")
	dumpPackets := flag.Bool("dump-packets", false, "log each UDP packet summary to stdout")
	uiFPS := flag.Int("ui-fps", 60, "maximum WebSocket frame rate (frames per second)")
	historySeconds := flag.Float64("history", 20, "capture history to keep per channel (seconds)")
	ingestQueue := flag.Int("ingest-q", 64, "UDP ingest queue length before processing")
	sampleRateFlag := flag.Float64("sample-rate", approxSampleRate, "per-channel ADC sample rate (samples per second)")
	fgAddr := flag.String("fg-addr", "192.168.10.2:6001", "UDP destination for function generator relay (MCU1 control port)")
	flag.Parse()

	if *historySeconds <= 0 {
		*historySeconds = 1
	}

	historySamples := int(*historySeconds * approxSampleRate)
	if historySamples < snapshotSamples {
		historySamples = snapshotSamples
	}
	const maxSamplesCap = 100_000_000 // ~100M samples per channel (~200 MB @ uint16)
	if historySamples > maxSamplesCap {
		historySamples = maxSamplesCap
	}
	if *ingestQueue < 1 {
		*ingestQueue = 1
	}
	sampleRate := *sampleRateFlag
	if sampleRate <= 0 {
		sampleRate = approxSampleRate
	}

	triggerCtl := newTriggerController()
	triggerCtl.SetSampleRate(sampleRate)

	fgRelay, err := newFGRelay(*fgAddr)
	if err != nil {
		log.Fatalf("fg relay init failed: %v", err)
	}

	hub := newWSHub(*uiFPS, triggerCtl, historySamples, *historySeconds, sampleRate, fgRelay)
	hub.Start()

	captureJobs := make(chan captureJob, *ingestQueue)
	go captureLoop(captureJobs, hub, triggerCtl)

	go func() {
		if err := runUDPReceiver(*udpListen, *dumpPackets, captureJobs); err != nil {
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

	fgStatus := "disabled"
	if fgRelay != nil && fgRelay.addr != nil {
		fgStatus = fgRelay.addr.String()
	}

	log.Printf("Serving UI at http://%s (UDP listener on %s, dump_packets=%v, fg_relay=%s)\n", addr, *udpListen, *dumpPackets, fgStatus)
	if err := http.ListenAndServe(*listen, mux); err != nil {
		log.Fatalf("http server failed: %v", err)
	}
}

func runUDPReceiver(listenAddr string, dumpPackets bool, jobs chan captureJob) error {
	udpAddr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return fmt.Errorf("resolve udp addr: %w", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("listen udp: %w", err)
	}
	defer conn.Close()
	defer close(jobs)

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

		samples, err := decodePacketSamples(header, payload)
		if err != nil {
			log.Printf("payload error from %s seq=%d: %v", remote, header.PacketSeq, err)
			continue
		}

		if dumpPackets {
			log.Println(summarizeSamples(header, samples))
		}

		job := captureJob{
			header:   header,
			samples:  samples,
			received: time.Now(),
		}

		select {
		case jobs <- job:
		default:
			// Channel full; drop oldest job and insert the new one.
			select {
			case <-jobs:
				log.Printf("capture queue full, dropping oldest job before seq=%d", header.PacketSeq)
			default:
			}
			jobs <- job
		}
	}
}

func captureLoop(queue <-chan captureJob, hub *wsHub, trigger *triggerController) {
	for job := range queue {
		shouldSend := true
		trigInfo := triggerInfo{
			Mode:  string(triggerModeAuto),
			Slope: string(triggerSlopeRising),
			Index: -1,
			State: "passthrough",
		}

		if trigger != nil {
			var trigErr error
			shouldSend, trigInfo, trigErr = trigger.Process(job.header, job.samples)
			if trigErr != nil {
				trigInfo.State = "error"
				log.Printf("trigger processing error seq=%d: %v", job.header.PacketSeq, trigErr)
			}
		}

		if !shouldSend {
			continue
		}

		delay := time.Since(job.received)
		hub.appendPacket(job.header, job.samples, trigInfo, delay)
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
	if sampleBytes != 1 {
		return packetHeader{}, nil, fmt.Errorf("sample bits %d not supported (only 8-bit is supported)", h.SampleBits)
	}

	origBytes := int(h.SamplesPerCh) * int(h.Channels) * sampleBytes
	expected := origBytes
	if len(payload) != expected {
		return packetHeader{}, nil, fmt.Errorf("payload mismatch: have %d, expected %d", len(payload), expected)
	}

	return h, payload, nil
}

func decodePacketSamples(h packetHeader, payload []byte) ([][]uint16, error) {
	sampleBytes := int(h.SampleBits / 8)
	if sampleBytes != 1 {
		return nil, fmt.Errorf("sample bits %d not supported (expected 8-bit)", h.SampleBits)
	}

	totalSamples := int(h.SamplesPerCh) * int(h.Channels)
	if totalSamples <= 0 {
		return nil, errors.New("no samples reported")
	}

	origBytes := totalSamples * sampleBytes
	if origBytes != len(payload) {
		return nil, fmt.Errorf("payload size mismatch: have %d need %d", len(payload), origBytes)
	}
	orig := payload[:origBytes]

	out := make([][]uint16, h.Channels)
	for ch := range out {
		out[ch] = make([]uint16, h.SamplesPerCh)
	}

	for i := 0; i < totalSamples; i++ {
		ch := i % int(h.Channels)
		idx := i / int(h.Channels)
		out[ch][idx] = uint16(orig[i])
	}

	return out, nil
}

func summarizeSamples(h packetHeader, samples [][]uint16) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("UDP seq=%d idx=%d ch=%d samples=%d flags=0x%X",
		h.PacketSeq, h.FirstSampleIdx, h.Channels, h.SamplesPerCh, h.Flags))

	for ch, data := range samples {
		preview := previewUint16(data, defaultPreview)
		sb.WriteString(fmt.Sprintf(" [ch%d first=%v]", ch, preview))
	}

	return sb.String()
}

func previewUint16(samples []uint16, limit int) []uint16 {
	if limit > len(samples) {
		limit = len(samples)
	}
	out := make([]uint16, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, samples[i])
	}
	return out
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
	if h.buffer == nil {
		return
	}

	evt, version, ok := h.buffer.snapshot(h.snapshotSize)
	if !ok || version == 0 {
		return
	}
	if version == h.lastBroadcast.Load() {
		return
	}

	if h.trigger != nil && evt.Trigger.Index == -1 {
		cfg := h.trigger.Config()
		ch := cfg.Channel
		if ch >= 0 && ch < len(evt.Samples) {
			bits := evt.SampleBits
			if bits == 0 {
				bits = 8
			}
			maxValue := (1 << bits) - 1
			level := uint16((int(cfg.Level) * maxValue) / 255)
			idx := findTriggerIndex(evt.Samples[ch], level, cfg.Slope)
			if idx >= 0 {
				evt.Trigger.Index = idx
				evt.Trigger.Channel = ch
				evt.Trigger.Level = level
				evt.Trigger.Mode = string(cfg.Mode)
				evt.Trigger.Slope = string(cfg.Slope)
				evt.Trigger.State = "postproc"
				evt.Trigger.Active = true
				evt.TriggerAbsIdx = evt.FirstSampleIdx + uint64(idx)
			}
		}
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

	data, err := json.Marshal(evt)
	if err != nil {
		log.Printf("ws marshal error: %v", err)
		return
	}

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
	var cmd wsCommand
	if err := json.Unmarshal(data, &cmd); err != nil {
		log.Printf("ws command decode error: %v", err)
		return
	}

	switch cmd.Cmd {
	case "set_trigger":
		if h.trigger == nil {
			return
		}
		update := triggerUpdate{
			Mode:      cmd.Mode,
			Slope:     cmd.Slope,
			Level:     cmd.Level,
			HoldoffUs: cmd.HoldoffUs,
			Channel:   cmd.Channel,
		}
		h.trigger.Update(update)
	case "arm_single":
		if h.trigger != nil {
			h.trigger.ArmSingle()
		}
	case "set_view":
		h.updateSnapshotSize(cmd.Samples)
	case "fg_send":
		if h.fg == nil {
			log.Printf("fg relay not configured; dropping fg_send")
			return
		}
		if err := h.fg.Send(cmd.Line); err != nil {
			log.Printf("fg relay send failed: %v", err)
		}
	default:
		log.Printf("ws unknown command: %s", cmd.Cmd)
	}
}
