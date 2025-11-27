#!/usr/bin/env python3
"""
Minimal UDP listener to confirm packets are arriving.
"""

import argparse
import socket
import time


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Simple UDP listener for debugging")
    parser.add_argument("--bind", default="0.0.0.0", help="interface/IP to bind (default: 0.0.0.0)")
    parser.add_argument("--port", type=int, default=5000, help="UDP port to listen on (default: 5000)")
    parser.add_argument(
        "--max-bytes",
        type=int,
        default=4096,
        help="max bytes to read per packet (default: 4096)",
    )
    return parser.parse_args()


def main() -> None:
    args = parse_args()

    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.bind((args.bind, args.port))
    print(f"listening on {args.bind}:{args.port} (Ctrl+C to stop)")

    packet_count = 0
    start_time = time.time()

    try:
        while True:
            data, addr = sock.recvfrom(args.max_bytes)
            packet_count += 1
            print(f"[{packet_count}] {addr[0]}:{addr[1]} -> {len(data)} bytes")
    except KeyboardInterrupt:
        elapsed = time.time() - start_time
        if elapsed > 0:
            rate = packet_count / elapsed
            print(f"\nstopped; received {packet_count} packets in {elapsed:.1f}s (~{rate:.1f} pkt/s)")
        else:
            print("\nstopped; no packets received")


if __name__ == "__main__":
    main()
