#include <stdint.h>
#include <stdio.h>

#include "eth.hh"
#include "user.hh"
#include "main.h"
#include "stm32f7xx_hal_def.h"

extern "C" {

extern UART_HandleTypeDef huart3;


int __io_putchar(int ch)
{
    uint8_t data = (uint8_t)ch;
    if (HAL_UART_Transmit(&huart3, &data, 1, HAL_MAX_DELAY) != HAL_OK)
    {
        return EOF;
    }

    return ch;
}

void UserCppInit(void)
{
    EthStreamReset(EthStreamGet());
}

void UserCppProcess(void)
{
    uint16_t idle_samples[kEthStreamChannels] = {0U};
    EthStreamQueueSamples(EthStreamGet(), idle_samples);
}

} /* extern "C" */
