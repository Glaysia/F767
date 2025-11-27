#!/usr/bin/env python3
"""
Lightweight UDP capture for STM32 ADC stream.
Receives packets, checks sequence/flags, and writes raw little-endian uint16
samples (interleaved ch0/ch1/ch2) to disk without any GUI work.
"""

import argparse
import socket
import struct
import time
from pathlib import Path


HEADER = struct.Struct("<IQHHHH")  # packet_seq, first_sample_idx, ch, samples/ch, flags, bits
DEFAULT_BIND = ("0.0.0.0", 5000)   # match MCU sender (192.168.10.2 -> 192.168.10.1:5000)


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="Capture ADC UDP packets to raw file")
    p.add_argument("-b", "--bind-ip", default=DEFAULT_BIND[0], help="IP to bind (default 0.0.0.0)")
    p.add_argument("-p", "--bind-port", type=int, default=DEFAULT_BIND[1], help="UDP port (default 5000)")
    p.add_argument("-o", "--output", type=Path, default=Path("adc_capture.bin"),
                   help="Output file for raw uint16 samples (interleaved)")
    p.add_argument("--no-write", action="store_true",
                   help="Do not write payload to disk (measure host RX ceiling)")
    p.add_argument("--rcvbuf", type=int, default=4 * 1024 * 1024,
                   help="Socket receive buffer bytes (increase to reduce drops)")
    p.add_argument("--limit-seconds", type=float, default=None,
                   help="Stop after this many seconds (default: run until Ctrl+C)")
    p.add_argument("--limit-packets", type=int, default=None,
                   help="Stop after this many packets")
    p.add_argument("--quiet", action="store_true", help="Reduce log output")
    return p.parse_args()


def main() -> None:
    args = parse_args()
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.bind((args.bind_ip, args.bind_port))
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_RCVBUF, args.rcvbuf)
    rcvbuf_applied = sock.getsockopt(socket.SOL_SOCKET, socket.SO_RCVBUF)
    sock.settimeout(1.0)

    buf = bytearray(2048)
    view = memoryview(buf)

    last_seq = None
    lost_packets = 0
    seq_jumps = 0
    drop_flags = 0
    packets = 0
    bytes_written = 0
    t_start = time.time()
    t_last_log = t_start

    args.output.parent.mkdir(parents=True, exist_ok=True)
    out = None if args.no_write else args.output.open("wb")

    if not args.quiet:
        dest = "discard" if args.no_write else f"writing to {args.output}"
        print(f"listening on {args.bind_ip}:{args.bind_port}, {dest}, SO_RCVBUF={rcvbuf_applied}")
    try:
        while True:
            if args.limit_seconds is not None and (time.time() - t_start) >= args.limit_seconds:
                break
            if args.limit_packets is not None and packets >= args.limit_packets:
                break

            try:
                nbytes, addr = sock.recvfrom_into(view)
            except socket.timeout:
                pass
            else:
                if nbytes < HEADER.size:
                    continue
                hdr = HEADER.unpack_from(view)
                packet_seq, first_idx, channels, samples_per_ch, flags, bits = hdr
                payload_bytes = channels * samples_per_ch * 2
                if (nbytes - HEADER.size) < payload_bytes:
                    continue

                if out is not None:
                    payload = view[HEADER.size:HEADER.size + payload_bytes]
                    out.write(payload)
                    bytes_written += payload_bytes
                packets += 1

                if last_seq is not None:
                    expected = (last_seq + 1) & 0xFFFFFFFF
                    if expected != packet_seq:
                        seq_jumps += 1
                        gap = (packet_seq - expected) & 0xFFFFFFFF
                        lost_packets += gap
                last_seq = packet_seq
                if (flags & 0x1) != 0:
                    drop_flags += 1

            now = time.time()
            if not args.quiet and (now - t_last_log) >= 1.0:
                elapsed = now - t_start
                rx_mbps = (bytes_written * 8.0 / 1e6) / elapsed if elapsed > 0 else 0.0
                pct_loss = (100.0 * lost_packets / (packets + lost_packets)
                            if (packets + lost_packets) > 0 else 0.0)
                print(f"{packets} pkts, {bytes_written/1e6:.2f} MB, "
                      f"{rx_mbps:.1f} Mbps, seq_jumps={seq_jumps}, "
                      f"lost={lost_packets} ({pct_loss:.2f}%), drop_flags={drop_flags}")
                t_last_log = now
    except KeyboardInterrupt:
        pass
    finally:
        if out is not None:
            out.close()

    if not args.quiet:
        elapsed = time.time() - t_start
        pct_loss = (100.0 * lost_packets / (packets + lost_packets)
                    if (packets + lost_packets) > 0 else 0.0)
        print(f"done: {packets} packets, {bytes_written/1e6:.2f} MB, "
              f"seq_jumps={seq_jumps}, lost={lost_packets} ({pct_loss:.2f}%), "
              f"drop_flags={drop_flags}, elapsed={elapsed:.1f}s")


if __name__ == "__main__":
    main()
