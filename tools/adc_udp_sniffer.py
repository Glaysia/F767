#!/usr/bin/env python3
import argparse
import socket
import struct
import time

HDR = struct.Struct("<IQHHHH")  # packet_seq, first_sample_idx, channels, samples_per_ch, flags, sample_bits

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--bind-ip", default="0.0.0.0")
    ap.add_argument("--port", type=int, default=5000)
    ap.add_argument("--timeout", type=float, default=1.0)
    args = ap.parse_args()

    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.bind((args.bind_ip, args.port))
    sock.settimeout(args.timeout)

    start = last = time.time()
    seq_prev = None
    pkts = gaps = drop_flags = badlen = 0
    minv, maxv = 255, 0

    print(f"Listening on {args.bind_ip}:{args.port} ...")
    while True:
        try:
            data, addr = sock.recvfrom(4096)
        except socket.timeout:
            print("timeout: no packets yet")
            continue

        now = time.time()
        pkts += 1
        if len(data) < HDR.size:
            badlen += 1
            continue

        packet_seq, first_idx, channels, samples_per_ch, flags, sample_bits = HDR.unpack_from(data)
        payload = data[HDR.size:]
        expected = channels * samples_per_ch
        if len(payload) != expected:
            badlen += 1
        if seq_prev is not None and packet_seq != ((seq_prev + 1) & 0xFFFFFFFF):
            gaps += 1
        seq_prev = packet_seq
        if flags & 0x1:
            drop_flags += 1
        if payload:
            pvmin, pvmax = min(payload), max(payload)
            minv, maxv = min(minv, pvmin), max(maxv, pvmax)

        if now - last >= 1.0:
            rate = pkts / (now - start)
            print(
                f"pkts={pkts} rate={rate:.0f}/s gaps={gaps} dropFlag={drop_flags} "
                f"badlen={badlen} ch={channels} spc={samples_per_ch} bits={sample_bits} "
                f"min/max={minv}/{maxv} from {addr[0]}"
            )
            last = now

if __name__ == "__main__":
    main()
