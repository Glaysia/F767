#!/usr/bin/env python3
import socket
import struct
import sys
from typing import Tuple

import matplotlib.pyplot as plt
import numpy as np

# Network config: match MCU sender (192.168.10.2 -> 192.168.10.1:5000)
BIND_IP = "0.0.0.0"
BIND_PORT = 5000

# Sampling params (per channel)
TIM5_TRIGGER_HZ = 1_255_800  # ~1.2558 MHz update rate
CHANNEL_COUNT = 3
FS_PER_CH = TIM5_TRIGGER_HZ / CHANNEL_COUNT

# Plot window
WINDOW_SAMPLES = 1024  # per channel

HEADER_FMT = "<IQHHHH"
HEADER_SIZE = struct.calcsize(HEADER_FMT)


def parse_packet(buf: bytes) -> Tuple[dict, np.ndarray]:
    if len(buf) < HEADER_SIZE:
        raise ValueError("short packet")
    packet_seq, first_sample_idx, channels, samples_per_ch, flags, _ = struct.unpack_from(
        HEADER_FMT, buf
    )
    payload = buf[HEADER_SIZE:]
    if channels == 0:
        raise ValueError("zero channels")
    expected_samples = channels * samples_per_ch
    if len(payload) < expected_samples * 2:
        raise ValueError("short payload")
    data = np.frombuffer(payload[: expected_samples * 2], dtype="<u2")
    frame = data.reshape((samples_per_ch, channels))
    hdr = {
        "packet_seq": packet_seq,
        "first_sample_idx": first_sample_idx,
        "channels": channels,
        "samples_per_ch": samples_per_ch,
        "flags": flags,
    }
    return hdr, frame


def main() -> None:
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.bind((BIND_IP, BIND_PORT))
    sock.settimeout(1.0)
    print(f"listening on {BIND_IP}:{BIND_PORT} ...")

    # Buffers for plotting
    t_buf = np.empty(0, dtype=np.float64)
    y_buf = np.empty((0, CHANNEL_COUNT), dtype=np.float64)
    last_seq = None

    fig, ax = plt.subplots()
    lines = [
        ax.plot([], [], label=f"CH{i}")[0] for i in range(CHANNEL_COUNT)
    ]
    ax.set_xlabel("time (s)")
    ax.set_ylabel("ADC code")
    ax.set_ylim(0, 4096)
    ax.legend(loc="upper right")

    def update_plot():
        for i, ln in enumerate(lines):
            ln.set_data(t_buf, y_buf[:, i])
        if t_buf.size:
            ax.set_xlim(t_buf[0], t_buf[-1])
        fig.canvas.draw_idle()

    try:
        while True:
            try:
                pkt, addr = sock.recvfrom(4096)
            except socket.timeout:
                plt.pause(0.01)
                continue

            try:
                hdr, frame = parse_packet(pkt)
            except ValueError as e:
                print(f"drop packet from {addr}: {e}", file=sys.stderr)
                continue

            if hdr["channels"] != CHANNEL_COUNT:
                print(f"unexpected channel count {hdr['channels']}", file=sys.stderr)
                continue

            if last_seq is not None and hdr["packet_seq"] != (last_seq + 1) % (1 << 32):
                print(f"seq jump {last_seq} -> {hdr['packet_seq']}", file=sys.stderr)
            last_seq = hdr["packet_seq"]

            t0 = hdr["first_sample_idx"] / FS_PER_CH
            ts = t0 + np.arange(hdr["samples_per_ch"]) / FS_PER_CH
            ys = frame.astype(np.float64)

            t_buf = np.concatenate([t_buf, ts])
            y_buf = np.vstack([y_buf, ys])
            if t_buf.size > WINDOW_SAMPLES:
                t_buf = t_buf[-WINDOW_SAMPLES:]
                y_buf = y_buf[-WINDOW_SAMPLES:, :]

            update_plot()
            plt.pause(0.001)
    except KeyboardInterrupt:
        pass


if __name__ == "__main__":
    main()
