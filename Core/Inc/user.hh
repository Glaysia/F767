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
 * Build lookup tables for the cat-head waveform (upper/lower halves).
 */
void UserBuildCatLuts(void);

/**
 * Start streaming the cat lookup tables to DAC channels 1 and 2.
 * @return HAL status aggregated from the DMA start calls.
 */
HAL_StatusTypeDef UserStartCatDac(void);

#ifdef __cplusplus
}
#endif
