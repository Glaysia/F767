#include <stdint.h>
#include <stddef.h>
#include <stdio.h>

#include "eth.hh"
#include "adc.hh"
#include "user.hh"
#include "main.h"
#include "lwip.h"


#include "stm32f7xx_hal_def.h"
#include "stm32f7xx_hal_gpio.h"
#include "stm32f7xx_hal_tim.h"

extern "C" {

extern UART_HandleTypeDef huart3;
extern TIM_HandleTypeDef htim3;


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
    
    HAL_TIM_Base_Start_IT(&htim3);
}

void UserCppProcess(void)
{
    AdcHandler::Process();
}

void HAL_TIM_PeriodElapsedCallback(TIM_HandleTypeDef *htim){
    if(htim == &htim3){
        MX_LWIP_Process();
        UserCppProcess();
    }
}

} /* extern "C" */
