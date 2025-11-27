# Agent Sync

## Coding Conventions
- Favor plain C syntax across the firmware; only minimal C++ features (e.g., struct methods) when absolutely necessary.
- Avoid namespaces, STL containers, and other high-level C++ abstractions in shared modules such as `Core/Src/user.cc`.

## Project Goal
- Build an STM32F767 (Nucleo-F767ZI) oscilloscope front end that streams ADC data over 100 Mbit/s Ethernet.
- Sample ADC1 channels IN0/IN3/IN4 in a scan sequence; target ~1.25 MHz sequence rate (~333 kSps per channel) to fit Ethernet throughput.

## Current Configuration / Done
- HAL ADC/TIM modules enabled. ADC1: 12-bit, scan ranks IN0/3/4, DMA circular, EOC on sequence, sampling time 28 cycles, external trigger edge rising on TIM5 TRGO.
- TIM5 master: TRGO=Update, PSC=0, ARR=85 (timer clk 108 MHz → ~1.2558 MHz), preload enabled, clock division /1.
- DMA2 Stream0 Channel0 for ADC1 (periph→mem, halfword, circular, mem increment, FIFO off); NVIC enabled.
- GPIO: PA0/PA3/PA4 analog for ADC; PF11 as EXTI11 (legacy trigger test); GPIOF clock enabled.
- ETH RX interrupt mode; ETH IRQ enabled. CMake driver list updated to include ADC/TIM. CubeMX metadata (`.mxproject`, `F767.ioc`) updated accordingly.

## Open Points / Next Steps
- In `main`, start peripherals: `HAL_ADC_Start_DMA(&hadc1, buf, len)` and `HAL_TIM_Base_Start(&htim5)`; remove/replace leftover TIM2 start if unused.
- Decide on `ContinuousConvMode` (currently ENABLE). For strict TIM5 pacing, consider DISABLE.
- Re-evaluate 28-cycle sampling time vs. front-end impedance; raise if source >1 kΩ or accuracy issues appear.
- Define DMA buffer layout and Ethernet packetization for 3-channel stream; keep within ~80 Mbit/s payload budget.
- Remove PF11 EXTI if unused to reduce spurious interrupts.
