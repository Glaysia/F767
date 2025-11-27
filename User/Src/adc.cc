#include "adc.hh"

extern "C" {
#include "stm32f7xx_hal.h"

extern ADC_HandleTypeDef hadc1;
extern TIM_HandleTypeDef htim5;
void Error_Handler(void);
}

static struct AdcSampleTriple g_latest_samples;
static uint16_t *g_adc_dma_buffer = NULL;
static size_t g_adc_dma_samples = 0U;

void AdcSampleTriple::Init(uint16_t *dma_buffer, size_t dma_samples)
{
    g_adc_dma_buffer = dma_buffer;
    g_adc_dma_samples = dma_samples;

    for (int i = 0; i < kAdcSampleChannels; ++i)
    {
        g_latest_samples.values[i] = 0U;
    }
}

void AdcSampleTriple::StartDma(void)
{
    if ((g_adc_dma_buffer == NULL) || (g_adc_dma_samples == 0U))
    {
        Error_Handler();
    }

    if (HAL_ADC_Start_DMA(&hadc1, (uint32_t *)g_adc_dma_buffer, g_adc_dma_samples) != HAL_OK)
    {
        Error_Handler();
    }
    if (HAL_TIM_Base_Start(&htim5) != HAL_OK)
    {
        Error_Handler();
    }
}

struct AdcSampleTriple AdcSampleTriple::GetLatest(void)
{
    return g_latest_samples;
}

AdcSampleTriple &AdcSampleTriple::Instance(void)
{
    return g_latest_samples;
}
