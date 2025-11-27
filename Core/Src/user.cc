/**
 * Example module that exposes C-callable entry points via user.hh.
 */

#include <math.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>

#include "user.hh"
#include "main.h"

static uint32_t g_process_counter = 0;

enum { kUserSineLutLength = 256 };
static const float kUserPi = 3.14159265358979323846f;
static uint32_t g_sine_lut[kUserSineLutLength];
static bool g_sine_lut_ready = false;

extern "C" void UserCppInit(void)
{
    g_process_counter = 0;
}

extern "C" {
extern UART_HandleTypeDef huart3;
extern DAC_HandleTypeDef hdac;
}

extern "C" void UserCppProcess(void)
{
    g_process_counter++;
}

extern "C" void UserBuild1HzSineLut(void)
{
    const float kFullScale = 4095.0f;
    const float kOffset = kFullScale / 2.0f;
    const float kAmplitude = kOffset * 0.95f; /* keep some headroom */
    const float step = (2.0f * kUserPi) / (float)kUserSineLutLength;

    for (size_t i = 0; i < kUserSineLutLength; i++)
    {
        const float angle = step * (float)i;
        const float value = kOffset + (kAmplitude * sinf(angle));
        g_sine_lut[i] = (uint32_t)lroundf(value);
    }

    g_sine_lut_ready = true;
}

extern "C" HAL_StatusTypeDef UserStart1HzSineDac(void)
{
    if (!g_sine_lut_ready)
    {
        UserBuild1HzSineLut();
    }

    // DAC expects 12-bit right aligned data when configured via CubeMX defaults.
    return HAL_DAC_Start_DMA(
        &hdac,
        DAC_CHANNEL_1,
        g_sine_lut,
        kUserSineLutLength,
        DAC_ALIGN_12B_R);
}

extern "C" int __io_putchar(int ch)
{
    uint8_t data = (uint8_t)ch;
    if (HAL_UART_Transmit(&huart3, &data, 1, HAL_MAX_DELAY) != HAL_OK)
    {
        return EOF;
    }

    return ch;
}
