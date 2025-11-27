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

void UserCppInit(void)
{
    g_process_counter = 0;
}

void UserCppProcess(void)
{
    g_process_counter++;
}

void UserBuildCatLuts(void)
{
    const size_t cat_samples = kUserCatLutLength / (1U + kUserCatRestRatio);
    const size_t rest_samples = kUserCatLutLength - cat_samples;
    const float step = 2.0f / (float)(cat_samples - 1U);
    const float top_scale = 1.7f;
    const float bottom_scale = 1.0f;

    for (size_t i = 0; i < cat_samples; i++)
    {
        float x = -1.0f + (step * (float)i);

        float circle = sqrtf(fmaxf(0.0f, 1.0f - (x * x)));
        float ears = 0.7f * (UserTriangle(x, -0.55f, 0.20f) + UserTriangle(x, 0.55f, 0.20f));
        float top_shape = circle + ears;
        float normalized_top = UserClamp(top_shape / top_scale, 0.0f, 1.0f);

        float chin_x = x * 0.85f;
        float chin_circle = sqrtf(fmaxf(0.0f, 1.0f - (chin_x * chin_x)));
        float chin_shape = powf(chin_circle, 1.35f);
        float normalized_bottom = UserClamp(chin_shape / bottom_scale, 0.0f, 1.0f);

        g_cat_top_lut[i] = (uint16_t)lroundf(normalized_top * 4095.0f);
        g_cat_bottom_lut[i] = (uint16_t)lroundf((1.0f - normalized_bottom) * 4095.0f);
    }

    for (size_t i = 0; i < rest_samples; i++)
    {
        size_t index = cat_samples + i;
        if (index >= kUserCatLutLength)
        {
            break;
        }

        g_cat_top_lut[index] = 0U;
        g_cat_bottom_lut[index] = 4095U;
    }

    g_cat_luts_ready = true;
}

HAL_StatusTypeDef UserStartCatDac(void)
{
    if (!g_cat_luts_ready)
    {
        UserBuildCatLuts();
    }

    HAL_StatusTypeDef status_top = HAL_DAC_Start_DMA(
        &hdac,
        DAC_CHANNEL_1,
        (uint32_t *)g_cat_top_lut,
        kUserCatLutLength,
        DAC_ALIGN_12B_R);

    HAL_StatusTypeDef status_bottom = HAL_DAC_Start_DMA(
        &hdac,
        DAC_CHANNEL_2,
        (uint32_t *)g_cat_bottom_lut,
        kUserCatLutLength,
        DAC_ALIGN_12B_R);

    return (status_top == HAL_OK && status_bottom == HAL_OK) ? HAL_OK : HAL_ERROR;
}

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
