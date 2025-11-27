#!/usr/bin/env python3
import socket

BIND_IP = "0.0.0.0"   # listen on all interfaces
BIND_PORT = 5000      # must match the MCU's UDP destination port


def main() -> None:
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.bind((BIND_IP, BIND_PORT))
    print(f"listening on {BIND_IP}:{BIND_PORT} ...")

    try:
        while True:
            data, addr = sock.recvfrom(4096)
            print(f"{addr} -> {len(data)} bytes: {data!r}")
    except KeyboardInterrupt:
        pass


if __name__ == "__main__":
    main()
