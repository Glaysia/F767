#pragma once

#include "stm32f7xx_hal.h"

#ifdef __cplusplus
extern "C" {
#endif

/**
 * Initializes any state shared between the C++ and C portions of the firmware.
 * Call this from C before invoking other functions declared in this header.
 */
void UserCppInit(void);

/**
 * Simple entry point implemented in C++ to demonstrate cross-language calls.
 * Safe to invoke from C code (e.g., main.c) whenever needed.
 */
void UserCppProcess(void);

/**
 * Build a 1 Hz sinewave lookup table for the DAC.
 */
void UserBuild1HzSineLut(void);

/**
 * Start pumping the sine LUT to DAC channel 1 using DMA.
 * @return HAL status from HAL_DAC_Start_DMA.
 */
HAL_StatusTypeDef UserStart1HzSineDac(void);

#ifdef __cplusplus
}
#endif
