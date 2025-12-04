# Agent Sync

## Coding Conventions
- Favor plain C syntax across the firmware; only minimal C++ features (e.g., struct methods) when absolutely necessary.
- Avoid namespaces, STL containers, and other high-level C++ abstractions in shared modules such as `Core/Src/user.cc`.

## Project Goal
- Build an STM32F767 (Nucleo-F767ZI) oscilloscope front end that streams ADC data over 100 Mbit/s Ethernet.
- Run ADC1 and ADC2 in parallel, each using a single channel (e.g., IN0 on ADC1, IN3 on ADC2) at 8-bit resolution around 2.4 MSa/s (≈19.2 Mbit/s per channel) so both streams fit comfortably within a 35 Mbit/s budget per channel.

## Current Configuration / Done
- HAL ADC/TIM modules enabled. ADC1: 12-bit, scan ranks IN0/3/4, DMA circular, EOC on sequence, sampling time 28 cycles, external trigger edge rising on TIM5 TRGO.
- TIM5 master: TRGO=Update, PSC=0, ARR=85 (timer clk 108 MHz → ~1.2558 MHz), preload enabled, clock division /1.
- DMA2 Stream0 Channel0 for ADC1 (periph→mem, halfword, circular, mem increment, FIFO off); NVIC enabled.
- GPIO: PA0/PA3/PA4 analog for ADC; PF11 as EXTI11 (legacy trigger test); GPIOF clock enabled.
- ETH RX interrupt mode; ETH IRQ enabled. CMake driver list updated to include ADC/TIM. CubeMX metadata (`.mxproject`, `F767.ioc`) updated accordingly.

## Open Points / Next Steps
- In `main`, start peripherals: `HAL_ADC_Start_DMA(&hadc1, buf, len)` / `HAL_ADC_Start_DMA(&hadc2, buf, len)` and `HAL_TIM_Base_Start(&htim5)`; remove/replace leftover TIM2 start if unused.
- Decide on `ContinuousConvMode` (currently ENABLE). For strict TIM5 pacing, consider DISABLE.
- Re-evaluate 28-cycle sampling time vs. front-end impedance; raise if source >1 kΩ or accuracy issues appear.
- Define DMA buffer layout and Ethernet packetization that sustain one 8-bit channel per ADC at ~2.4 MSa/s (~19.2 Mbit/s) without overrunning the 35 Mbit/s-per-channel budget.
- Remove PF11 EXTI if unused to reduce spurious interrupts.
