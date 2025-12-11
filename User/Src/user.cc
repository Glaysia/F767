#include <stdint.h>
#include <stddef.h>
#include <stdio.h>

#include "eth.hh"
#include "adc.hh"
#include "user.hh"
#include "main.h"
#include "fg_relay.h"


#include "stm32f7xx_hal_def.h"
#include "stm32f7xx_hal_gpio.h"

extern "C" {

extern UART_HandleTypeDef huart3;
extern UART_HandleTypeDef huart4;


int __io_putchar(int ch)
{
    uint8_t data = (uint8_t)ch;
    if (HAL_UART_Transmit(&huart3, &data, 1, HAL_MAX_DELAY) != HAL_OK)
    {
        return EOF;
    }

    return ch;
}

void UserCppInit(uint16_t *adc_dma_buffer, size_t adc_dma_samples)
{
    AdcHandler::Init(adc_dma_buffer, adc_dma_samples);
    AdcHandler::StartDma();
    EthStream::Instance().Reset();
    FgRelay_Init();
}

void UserCppProcess(void)
{
    AdcHandler::Process();
    FgRelay_Process();
}

} /* extern "C" */
