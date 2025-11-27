/**
 * Example C++ module that exposes C-callable entry points via user.hh.
 * Use this file when adding modern C++ code to the firmware.
 */

#include <array>
#include <cmath>
#include <cstdint>
#include <cstdio>

#include "user.hh"
#include "main.h"

namespace
{
std::uint32_t g_process_counter = 0;

constexpr std::size_t kLutLength = 256;
constexpr float kPi = 3.14159265358979323846f;
std::array<std::uint32_t, kLutLength> g_sine_lut{};
bool g_sine_lut_ready = false;
}

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
    ++g_process_counter;
}

extern "C" void UserBuild1HzSineLut(void)
{
    constexpr float kFullScale = 4095.0f;
    constexpr float kOffset = kFullScale / 2.0f;
    constexpr float kAmplitude = kOffset * 0.95f; // keep some headroom
    const float step = (2.0f * kPi) / static_cast<float>(kLutLength);

    for (std::size_t i = 0; i < kLutLength; ++i)
    {
        const float angle = step * static_cast<float>(i);
        const float value = kOffset + (kAmplitude * std::sin(angle));
        g_sine_lut[i] = static_cast<std::uint32_t>(std::lround(value));
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
        g_sine_lut.data(),
        g_sine_lut.size(),
        DAC_ALIGN_12B_R);
}

extern "C" int __io_putchar(int ch)
{
    std::uint8_t data = static_cast<std::uint8_t>(ch);
    if (HAL_UART_Transmit(&huart3, &data, 1, HAL_MAX_DELAY) != HAL_OK)
    {
        return EOF;
    }

    return ch;
}
