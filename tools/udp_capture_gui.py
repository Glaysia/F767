#!/usr/bin/env python3
"""
Live UDP capture/plot for the STM32 ADC stream.

Matches EthPacketHeader (little-endian):
uint32 packet_seq, uint64 first_sample_idx, uint16 channels, uint16 samples_per_ch,
uint16 flags, uint16 sample_bits. Payload is interleaved uint16 samples.
"""

import argparse
import socket
import struct
import sys
from dataclasses import dataclass
from typing import Tuple

import matplotlib.pyplot as plt
import numpy as np

HEADER_STRUCT = struct.Struct("<IQHHHH")
HEADER_SIZE = HEADER_STRUCT.size

DEFAULT_BIND_IP = "0.0.0.0"
DEFAULT_BIND_PORT = 5000
DEFAULT_RCVBUF = 4 * 1024 * 1024
DEFAULT_TIMEOUT = 0.5


@dataclass
class PacketHeader:
    packet_seq: int
    first_sample_idx: int
    channels: int
    samples_per_ch: int
    flags: int
    sample_bits: int

    @classmethod
    def parse(cls, buf: bytes) -> "PacketHeader":
        if len(buf) < HEADER_SIZE:
            raise ValueError("short header")
        return cls(*HEADER_STRUCT.unpack_from(buf))


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Live plot of MCU ADC UDP stream")
    parser.add_argument("--bind-ip", default=DEFAULT_BIND_IP, help="IP/interface to bind")
    parser.add_argument("--bind-port", type=int, default=DEFAULT_BIND_PORT, help="UDP port to bind")
    parser.add_argument("--channels", type=int, default=2, help="Expected channel count")
    parser.add_argument(
        "--fs",
        type=float,
        default=1_255_800.0,
        help="Per-channel sample rate (Hz); used for time axis",
    )
    parser.add_argument(
        "--window", type=int, default=2048, help="Samples per channel to retain in plot"
    )
    parser.add_argument("--timeout", type=float, default=DEFAULT_TIMEOUT, help="Socket timeout seconds")
    parser.add_argument("--rcvbuf", type=int, default=DEFAULT_RCVBUF, help="SO_RCVBUF size")
    parser.add_argument(
        "--ylim",
        type=int,
        default=None,
        help="Force y-axis max (default: use sample_bits from stream)",
    )
    return parser.parse_args()


def create_socket(args: argparse.Namespace) -> socket.socket:
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_RCVBUF, args.rcvbuf)
    sock.bind((args.bind_ip, args.bind_port))
    sock.settimeout(args.timeout)
    return sock


def parse_packet(data: bytes, expected_channels: int) -> Tuple[PacketHeader, np.ndarray]:
    if len(data) < HEADER_SIZE:
        raise ValueError("short packet")

    hdr = PacketHeader.parse(data)
    if hdr.channels != expected_channels:
        raise ValueError(f"unexpected channel count {hdr.channels}")

    payload_len = hdr.channels * hdr.samples_per_ch * 2  # stream packs uint16 samples
    if len(data) - HEADER_SIZE < payload_len:
        raise ValueError("short payload")

    frame = np.frombuffer(
        data, dtype="<u2", offset=HEADER_SIZE, count=payload_len // 2
    ).reshape((hdr.samples_per_ch, hdr.channels))
    return hdr, frame


def main() -> None:
    args = parse_args()
    sock = create_socket(args)
    print(
        f"listening on {args.bind_ip}:{args.bind_port}, "
        f"expect {args.channels} ch @ {args.fs:.3f} Hz"
    )

    t_buf = np.empty(0, dtype=np.float64)
    y_buf = np.empty((0, args.channels), dtype=np.float64)
    last_seq = None

    plt.ion()
    fig, ax = plt.subplots()
    lines = [ax.plot([], [], label=f"CH{i}")[0] for i in range(args.channels)]
    ax.set_xlabel("time (s)")
    ax.set_ylabel("ADC code")
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
                hdr, frame = parse_packet(pkt, args.channels)
            except ValueError as e:
                print(f"drop {addr}: {e}", file=sys.stderr)
                continue

            if last_seq is not None and hdr.packet_seq != ((last_seq + 1) & 0xFFFFFFFF):
                print(f"seq jump {last_seq} -> {hdr.packet_seq}", file=sys.stderr)
            last_seq = hdr.packet_seq

            if args.ylim is not None:
                ax.set_ylim(0, args.ylim)
            elif hdr.sample_bits > 0:
                ax.set_ylim(0, (1 << hdr.sample_bits))

            t0 = hdr.first_sample_idx / args.fs
            ts = t0 + np.arange(hdr.samples_per_ch, dtype=np.float64) / args.fs
            ys = frame.astype(np.float64)

            if hdr.flags:
                print(f"packet {hdr.packet_seq} flags=0x{hdr.flags:x}", file=sys.stderr)

            t_buf = np.concatenate([t_buf, ts])
            y_buf = np.vstack([y_buf, ys])
            if t_buf.size > args.window:
                t_buf = t_buf[-args.window :]
                y_buf = y_buf[-args.window :, :]

            update_plot()
            plt.pause(0.001)
    except KeyboardInterrupt:
        pass
    finally:
        sock.close()


if __name__ == "__main__":
    main()
