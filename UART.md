# UART Function Generator Interface

## Serial Settings
- Port: `UART4` (PC10 → TX, PC11 → RX), interrupt-driven RX.
- Baud: `115200`, data bits: `8`, parity: `None`, stop bits: `1`, flow control: `None`.
- Firmware prints:
  ````
  Function Generator Started
  Type 'H' for help
  ````
  on startup.
- Commands are buffered until `\r` or `\n`.

### Network Relay
- MCU1 수신 포트: UDP `6001` (`fg-addr` 플래그로 변경 가능). 페이로드는 위 명령 문자열을 그대로 담아 전송하면 MCU1이 UART4로 릴레이한다.

## Command Summary
| Command    | Description                                 | Response (success)                | Notes                      |
|------------|---------------------------------------------|----------------------------------|----------------------------|
| `0`, `1`, `2`, `3` | Quick waveform select (0=SINE, 1=SQUARE, 2=TRIANGLE, 3=SAWTOOTH) | `Waveform: <name>`              | No prefix, single digit.   |
| `W<0-3>`   | Explicit waveform select                    | `OK: Wave=<name>`                | Error if digit out of range. |
| `F<Hz>`    | Set frequency (100–100000 Hz)               | `OK: Freq=<value> Hz`            | Uses integer Hz; errors outside range. |
| `A<val>`   | Set amplitude (DAC counts 0–4095)           | `OK: Amplitude=<value> (0..4095)`| Errors outside range.      |
| `D`        | Display current configuration               | `Freq:<Hz> Hz | Waveform:<name> | Amplitude:<val>` | Read-only status.          |
| `H`        | Help                                        | Prints full command list         | Same text as startup tip.  |

## Default State
- Frequency: `1000 Hz`
- Waveform: `SINE`
- Amplitude: `4095` (full-scale)
- Typing `H` after connecting shows allowed commands.

## Usage Tips
1. Configure your terminal to `115200/8/N/1`, no flow control.
2. Connect PA0→RX of PC, PC11→TX of PC (crossed).
3. Type commands followed immediately by the numeric argument (no spaces), press Enter.
4. Wait for the acknowledgment string before issuing the next command.
5. If output behaves oddly, send `D` to confirm current parameters or `H` to re-read the command list.
