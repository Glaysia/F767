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
#include "stm32f7xx_hal_def.h"

static uint32_t g_process_counter = 0;

enum { kUserCatLutLength = 256 };
enum { kUserCatRestRatio = 2 };
static uint16_t g_cat_top_lut[kUserCatLutLength];
static uint16_t g_cat_bottom_lut[kUserCatLutLength];
static bool g_cat_luts_ready = false;

static float UserClamp(float value, float min_value, float max_value)
{
    if (value < min_value)
    {
        return min_value;
    }
    if (value > max_value)
    {
        return max_value;
    }

    return value;
}

static float UserTriangle(float x, float center, float half_width)
{
    float distance = fabsf(x - center);
    if (distance >= half_width)
    {
        return 0.0f;
    }

    return 1.0f - (distance / half_width);
}

extern "C" {

extern UART_HandleTypeDef huart3;
extern DAC_HandleTypeDef hdac;

int __io_putchar(int ch)
{
    uint8_t data = (uint8_t)ch;
    if (HAL_UART_Transmit(&huart3, &data, 1, HAL_MAX_DELAY) != HAL_OK)
    {
        return EOF;
    }

    return ch;
}

} /* extern "C" */
