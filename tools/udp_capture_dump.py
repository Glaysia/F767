#!/usr/bin/env python3
"""
Headless UDP capture utility for the STM32 ADC stream.

The MCU emits packets with the header defined in EthPacketHeader (little-endian).
This tool listens on the specified interface, validates each packet, optionally
stores the interleaved uint16 samples to disk, and reports throughput/loss
statistics so issues on either end are easy to spot.
"""

import argparse
import socket
import struct
import sys
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Optional

HEADER_FMT = "<IQHHHH"
HEADER = struct.Struct(HEADER_FMT)
HEADER_SIZE = HEADER.size

DEFAULT_BIND_IP = "0.0.0.0"
DEFAULT_BIND_PORT = 5000
DEFAULT_BUFFER_SIZE = 8192
DEFAULT_TIMEOUT = 0.5
DEFAULT_RCVBUF = 4 * 1024 * 1024


@dataclass
class PacketHeader:
    packet_seq: int
    first_sample_idx: int
    channels: int
    samples_per_ch: int
    flags: int
    sample_bits: int

    @classmethod
    def parse(cls, view: memoryview) -> "PacketHeader":
        if len(view) < HEADER_SIZE:
            raise ValueError("short header")
        return cls(*HEADER.unpack_from(view))


@dataclass
class CaptureStats:
    packets: int = 0
    payload_bytes: int = 0
    seq_jumps: int = 0
    lost_packets: int = 0
    drop_flags: int = 0
    short_packets: int = 0
    malformed: int = 0
    last_seq: Optional[int] = None

    def account_packet(self, header: PacketHeader, payload_len: int) -> None:
        self.packets += 1
        self.payload_bytes += payload_len
        if self.last_seq is not None:
            expected = (self.last_seq + 1) & 0xFFFFFFFF
            if header.packet_seq != expected:
                self.seq_jumps += 1
                gap = (header.packet_seq - expected) & 0xFFFFFFFF
                self.lost_packets += gap
        self.last_seq = header.packet_seq
        if (header.flags & 0x1) != 0:
            self.drop_flags += 1

    def summary(self, elapsed: float) -> str:
        mb_written = self.payload_bytes / 1e6
        mbps = (self.payload_bytes * 8.0 / 1e6) / elapsed if elapsed > 0 else 0.0
        total_rx = self.packets + self.lost_packets
        loss_pct = (100.0 * self.lost_packets / total_rx) if total_rx else 0.0
        return (
            f"{self.packets} pkts, {mb_written:.2f} MB payload, "
            f"{mbps:.1f} Mbps, seq_jumps={self.seq_jumps}, "
            f"lost={self.lost_packets} ({loss_pct:.2f}%), "
            f"drop_flags={self.drop_flags}, short={self.short_packets}, "
            f"malformed={self.malformed}"
        )


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Capture MCU ADC UDP stream to disk")
    parser.add_argument("--bind-ip", default=DEFAULT_BIND_IP, help="IP/interface to bind")
    parser.add_argument(
        "--bind-port", type=int, default=DEFAULT_BIND_PORT, help="UDP port to bind"
    )
    parser.add_argument(
        "-o",
        "--output",
        type=Path,
        default=Path("adc_capture.bin"),
        help="Output file for raw uint16 samples",
    )
    parser.add_argument(
        "--no-write", action="store_true", help="Do not write payload (stats only)"
    )
    parser.add_argument(
        "--expect-channels",
        type=int,
        default=None,
        help="Drop packets whose channel count differs (default: accept any)",
    )
    parser.add_argument(
        "--limit-packets", type=int, default=None, help="Stop after this many packets"
    )
    parser.add_argument(
        "--limit-seconds",
        type=float,
        default=None,
        help="Stop after this many seconds",
    )
    parser.add_argument(
        "--limit-bytes",
        type=int,
        default=None,
        help="Stop after writing this many payload bytes",
    )
    parser.add_argument(
        "--buffer-size",
        type=int,
        default=DEFAULT_BUFFER_SIZE,
        help="Receive buffer size passed to recvfrom_into",
    )
    parser.add_argument(
        "--rcvbuf",
        type=int,
        default=DEFAULT_RCVBUF,
        help="SO_RCVBUF value (kernel receive queue)",
    )
    parser.add_argument(
        "--stats-interval",
        type=float,
        default=1.0,
        help="Seconds between progress prints",
    )
    parser.add_argument("--quiet", action="store_true", help="Suppress periodic logs")
    parser.add_argument(
        "--timeout",
        type=float,
        default=DEFAULT_TIMEOUT,
        help="Socket timeout seconds",
    )
    return parser.parse_args()


def create_socket(args: argparse.Namespace) -> socket.socket:
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.bind((args.bind_ip, args.bind_port))
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_RCVBUF, args.rcvbuf)
    sock.settimeout(args.timeout)
    return sock


def should_stop(args: argparse.Namespace, stats: CaptureStats, elapsed: float) -> bool:
    if args.limit_seconds is not None and elapsed >= args.limit_seconds:
        return True
    if args.limit_packets is not None and stats.packets >= args.limit_packets:
        return True
    if args.limit_bytes is not None and stats.payload_bytes >= args.limit_bytes:
        return True
    return False


def capture_loop(sock: socket.socket, args: argparse.Namespace) -> CaptureStats:
    buf = bytearray(max(args.buffer_size, HEADER_SIZE + 2))
    view = memoryview(buf)
    stats = CaptureStats()
    args.output.parent.mkdir(parents=True, exist_ok=True)
    out_file = None if args.no_write else args.output.open("wb")
    start = time.time()
    next_log = start + args.stats_interval

    if not args.quiet:
        dest = "discard" if args.no_write else f"writing to {args.output}"
        rcvbuf = sock.getsockopt(socket.SOL_SOCKET, socket.SO_RCVBUF)
        print(
            f"Listening on {args.bind_ip}:{args.bind_port}, {dest}, "
            f"SO_RCVBUF={rcvbuf}"
        )

    try:
        while True:
            now = time.time()
            elapsed = now - start
            if should_stop(args, stats, elapsed):
                break

            try:
                nbytes, _ = sock.recvfrom_into(view)
            except socket.timeout:
                # idle, just drop to log/update
                pass
            except OSError as exc:
                print(f"socket error: {exc}", file=sys.stderr)
                break
            else:
                if nbytes < HEADER_SIZE:
                    stats.malformed += 1
                    continue

                try:
                    header = PacketHeader.parse(view[:HEADER_SIZE])
                except ValueError:
                    stats.malformed += 1
                    continue

                if args.expect_channels is not None and header.channels != args.expect_channels:
                    stats.malformed += 1
                    continue

                payload_len = header.channels * header.samples_per_ch * 2
                if payload_len == 0:
                    stats.malformed += 1
                    continue

                if (nbytes - HEADER_SIZE) < payload_len:
                    stats.short_packets += 1
                    continue

                if out_file is not None:
                    payload = view[HEADER_SIZE : HEADER_SIZE + payload_len]
                    out_file.write(payload)

                stats.account_packet(header, payload_len)

            if not args.quiet and now >= next_log:
                print(stats.summary(max(now - start, 1e-6)))
                next_log = now + args.stats_interval
    finally:
        if out_file is not None:
            out_file.close()

    if not args.quiet:
        elapsed = max(time.time() - start, 1e-6)
        print(f"Done: {stats.summary(elapsed)} (elapsed {elapsed:.1f}s)")

    return stats


def main() -> None:
    args = parse_args()
    sock = create_socket(args)
    try:
        capture_loop(sock, args)
    finally:
        sock.close()


if __name__ == "__main__":
    main()
