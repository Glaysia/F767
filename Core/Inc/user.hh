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



#ifdef __cplusplus
}
#endif
