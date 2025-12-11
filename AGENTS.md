# Agent Sync

## Coding Conventions
- Favor plain C syntax across the firmware; only minimal C++ features (e.g., struct methods) when absolutely necessary.
- Avoid namespaces, STL containers, and other high-level C++ abstractions in shared modules such as `Core/Src/user.cc`.

## Project Goal
- Build an STM32F767 (Nucleo-F767ZI) oscilloscope front end that streams ADC data over 100 Mbit/s Ethernet.
- Run ADC1 and ADC2 in parallel, each using a single channel (e.g., IN0 on ADC1, IN3 on ADC2) at 8-bit resolution around 2.4 MSa/s (≈19.2 Mbit/s per channel) so both streams fit comfortably within a 35 Mbit/s budget per channel.

## Current Configuration / Done
- HAL ADC/TIM modules enabled. ADC1 (IN0) and ADC3 (IN6) set to 8-bit, single-channel, DMA circular, EOC on sequence, sampling time 3 cycles, external trigger edge rising on TIM3 CC4.
- TIM3: PSC=0, ARR=85 (~1.2558 MHz), TRGO=Update, CH4 in OC timing mode (pulse=0). `AdcHandler::StartDma` starts ADC1/ADC3 DMAs plus `HAL_TIM_Base_Start` and `HAL_TIM_OC_Start` on CH4 to generate CC4 triggers.
- TIM5 still configured (PSC=0, ARR=85, TRGO=Update, preload enabled) but not driving ADCs.
- DMA2 Stream0 Channel0 for ADC1 and DMA2 Stream1 Channel2 for ADC3 (periph→mem, halfword, circular, mem increment, FIFO off); NVIC enabled.
- GPIO: PA0 analog for ADC1_IN0; PF8 analog for ADC3_IN6; PF11 as EXTI11 (legacy trigger test); GPIOF clock enabled.
- ETH RX interrupt mode; ETH IRQ enabled. CMake driver list updated to include ADC/TIM. CubeMX metadata (`.mxproject`, `F767.ioc`) updated accordingly.

## Open Points / Next Steps
- Decide whether to keep TIM3 CC4 triggering or move to TIM5 TRGO; if switching, change ADC external trigger and start the matching timer.
- `ContinuousConvMode` currently DISABLE for both ADCs; keep if timer-paced, enable only if needed for a different trigger model.
- Re-evaluate 3-cycle sampling time vs. front-end impedance; raise if source >1 kΩ or accuracy issues appear.
- Define DMA buffer layout and Ethernet packetization that sustain one 8-bit channel per ADC at ~2.4 MSa/s (~19.2 Mbit/s) without overrunning the 35 Mbit/s-per-channel budget.
- Remove PF11 EXTI if unused to reduce spurious interrupts.
