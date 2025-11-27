#pragma once

#include "stm32f7xx_hal.h"
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * Initializes any state shared between the C++ and C portions of the firmware.
 * Call this from C before invoking other functions declared in this header.
 */
void UserCppInit(uint16_t *adc_dma_buffer, size_t adc_dma_samples);
void UserCppProcess(void);



#ifdef __cplusplus
}
#endif
