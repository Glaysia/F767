#!/usr/bin/env python3
"""
Lightweight UDP sniffer to confirm packets are arriving, regardless of protocol.
"""

import argparse
import socket
import time
from typing import Tuple


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Minimal UDP sniffer")
    parser.add_argument("--bind", default="0.0.0.0", help="IP/interface to bind (default: 0.0.0.0)")
    parser.add_argument("--port", type=int, default=5000, help="UDP port to listen on (default: 5000)")
    parser.add_argument("--max-bytes", type=int, default=4096, help="Max bytes to read per packet")
    parser.add_argument("--rcvbuf", type=int, default=4 * 1024 * 1024, help="SO_RCVBUF size")
    parser.add_argument(
        "--hex",
        type=int,
        default=16,
        help="Print this many bytes of payload as hex (0 to disable)",
    )
    parser.add_argument(
        "--stats",
        type=float,
        default=1.0,
        help="Seconds between throughput prints (set 0 to disable)",
    )
    return parser.parse_args()


def hexdump(buf: bytes, limit: int) -> str:
    if limit <= 0:
        return ""
    shown = buf[:limit]
    return " ".join(f"{b:02x}" for b in shown)


def rate_summary(pkts: int, bytes_rx: int, elapsed: float) -> str:
    mbps = (bytes_rx * 8.0 / 1e6) / elapsed if elapsed > 0 else 0.0
    pps = pkts / elapsed if elapsed > 0 else 0.0
    return f"{pkts} pkts, {bytes_rx} bytes, {mbps:.2f} Mbps, {pps:.1f} pkt/s"


def main() -> None:
    args = parse_args()
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_RCVBUF, args.rcvbuf)
    sock.bind((args.bind, args.port))
    sock.settimeout(0.5)

    print(f"listening on {args.bind}:{args.port}, max_bytes={args.max_bytes}, SO_RCVBUF={args.rcvbuf}")

    packets = 0
    bytes_rx = 0
    start = time.time()
    next_stats = start + args.stats if args.stats > 0 else None

    try:
        while True:
            try:
                data, addr = sock.recvfrom(args.max_bytes)
            except socket.timeout:
                pass
            else:
                packets += 1
                bytes_rx += len(data)
                prefix = hexdump(data, args.hex)
                hex_part = f" hex[{prefix}]" if prefix else ""
                print(f"[{packets}] {addr[0]}:{addr[1]} -> {len(data)} bytes{hex_part}")

            now = time.time()
            if next_stats is not None and now >= next_stats:
                elapsed = max(now - start, 1e-6)
                print(f"stats: {rate_summary(packets, bytes_rx, elapsed)}")
                next_stats = now + args.stats
    except KeyboardInterrupt:
        pass
    finally:
        sock.close()
        elapsed = max(time.time() - start, 1e-6)
        print(f"stopped after {elapsed:.1f}s: {rate_summary(packets, bytes_rx, elapsed)}")


if __name__ == "__main__":
    main()
