#pragma once

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

#ifdef __cplusplus
}
#endif
