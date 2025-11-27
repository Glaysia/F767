/**
 * Example C++ module that exposes C-callable entry points via user.hh.
 * Use this file when adding modern C++ code to the firmware.
 */

#include <cstdint>
#include <cstdio>

#include "user.hh"
#include "main.h"

namespace
{
std::uint32_t g_process_counter = 0;
}

extern "C" void UserCppInit(void)
{
    g_process_counter = 0;
}

extern "C" {
extern UART_HandleTypeDef huart3;
}

extern "C" void UserCppProcess(void)
{
    ++g_process_counter;
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
