#include "adc.hh"

#include <string.h>

#include "eth.hh"

extern "C" {
#include "stm32f7xx_hal.h"

extern ADC_HandleTypeDef hadc1;
extern ADC_HandleTypeDef hadc2;
extern TIM_HandleTypeDef htim5;
void Error_Handler(void);
}

enum
{
    kAdcFrameSamples = kEthStreamFrameCapacity * kEthStreamChannels,
    kAdcFrameQueueDepth = 4
};

struct AdcFrame
{
    uint16_t samples[kAdcFrameSamples];
    size_t sample_count;
    uint16_t flags;
};

static uint16_t *g_adc1_dma_buffer = NULL;
static uint16_t *g_adc2_dma_buffer = NULL;
static size_t g_adc_samples_per_adc = 0U;
static size_t g_samples_per_channel = 0U;

static const uint16_t *g_pending_half[2][2];

static AdcFrame g_frame_queue[kAdcFrameQueueDepth];
static volatile size_t g_frame_read = 0U;
static volatile size_t g_frame_write = 0U;
static volatile uint16_t g_drop_latch = 0U;

static void AdcHandler_HandleDmaBlock(size_t adc_index, size_t base_index);
static void AdcHandler_TryEnqueue(size_t half_index);

void AdcHandler::Init(uint16_t *adc1_buffer, uint16_t *adc2_buffer, size_t samples_per_adc)
{
    g_adc1_dma_buffer = adc1_buffer;
    g_adc2_dma_buffer = adc2_buffer;
    g_adc_samples_per_adc = samples_per_adc;

    if ((g_adc1_dma_buffer == NULL) || (g_adc2_dma_buffer == NULL) || (g_adc_samples_per_adc == 0U) || ((g_adc_samples_per_adc % 2U) != 0U))
    {
        Error_Handler();
    }

    g_samples_per_channel = g_adc_samples_per_adc / 2U;
    if (g_samples_per_channel != kEthStreamFrameCapacity)
    {
        Error_Handler();
    }

    memset((void *)g_pending_half, 0, sizeof(g_pending_half));
    g_frame_read = 0U;
    g_frame_write = 0U;
    g_drop_latch = 0U;
}

void AdcHandler::StartDma(void)
{
    if ((g_adc1_dma_buffer == NULL) || (g_adc2_dma_buffer == NULL) || (g_adc_samples_per_adc == 0U))
    {
        Error_Handler();
    }

    if (HAL_ADC_Start_DMA(&hadc1, (uint32_t *)g_adc1_dma_buffer, g_adc_samples_per_adc) != HAL_OK)
    {
        Error_Handler();
    }
    if (HAL_ADC_Start_DMA(&hadc2, (uint32_t *)g_adc2_dma_buffer, g_adc_samples_per_adc) != HAL_OK)
    {
        Error_Handler();
    }
    if (HAL_TIM_Base_Start(&htim5) != HAL_OK)
    {
        Error_Handler();
    }
}

void AdcHandler::Process(void)
{
    while (g_frame_read != g_frame_write)
    {
        AdcFrame &frame = g_frame_queue[g_frame_read];
        const bool sent = EthStream::Instance().SendFrame(frame.samples, frame.sample_count, frame.flags);
        if (!sent)
        {
            g_drop_latch = 1U;
        }

        g_frame_read = (g_frame_read + 1U) % kAdcFrameQueueDepth;
    }
}

static void AdcHandler_HandleDmaBlock(size_t adc_index, size_t base_index)
{
    if ((adc_index > 1U) || (g_samples_per_channel == 0U) || (g_adc_samples_per_adc == 0U))
    {
        return;
    }

    if ((base_index + g_samples_per_channel) > g_adc_samples_per_adc)
    {
        return;
    }

    const size_t half_index = (base_index >= g_samples_per_channel) ? 1U : 0U;
    const uint16_t *base_ptr = (adc_index == 0U) ? &g_adc1_dma_buffer[base_index] : &g_adc2_dma_buffer[base_index];
    if (g_pending_half[adc_index][half_index] != NULL)
    {
        g_drop_latch = 1U;
    }
    g_pending_half[adc_index][half_index] = base_ptr;
    AdcHandler_TryEnqueue(half_index);
}

static void AdcHandler_TryEnqueue(size_t half_index)
{
    if (half_index > 1U)
    {
        return;
    }

    const uint16_t *src0 = g_pending_half[0][half_index];
    const uint16_t *src1 = g_pending_half[1][half_index];
    if ((src0 == NULL) || (src1 == NULL))
    {
        return;
    }

    size_t next_write = (g_frame_write + 1U) % kAdcFrameQueueDepth;
    if (next_write == g_frame_read)
    {
        g_drop_latch = 1U;
        g_pending_half[0][half_index] = NULL;
        g_pending_half[1][half_index] = NULL;
        return;
    }

    AdcFrame &frame = g_frame_queue[g_frame_write];
    for (size_t i = 0U; i < g_samples_per_channel; ++i)
    {
        const size_t dst_idx = i * kEthStreamChannels;
        frame.samples[dst_idx] = src0[i];
        frame.samples[dst_idx + 1U] = src1[i];
    }

    frame.sample_count = g_samples_per_channel * kEthStreamChannels;
    frame.flags = g_drop_latch;
    g_drop_latch = 0U;

    g_frame_write = next_write;

    g_pending_half[0][half_index] = NULL;
    g_pending_half[1][half_index] = NULL;
}

extern "C" void HAL_ADC_ConvHalfCpltCallback(ADC_HandleTypeDef *hadc)
{
    if (hadc == &hadc1)
    {
        AdcHandler_HandleDmaBlock(0U, 0U);
    }
    else if (hadc == &hadc2)
    {
        AdcHandler_HandleDmaBlock(1U, 0U);
    }
}

extern "C" void HAL_ADC_ConvCpltCallback(ADC_HandleTypeDef *hadc)
{
    if (hadc == &hadc1)
    {
        AdcHandler_HandleDmaBlock(0U, g_samples_per_channel);
    }
    else if (hadc == &hadc2)
    {
        AdcHandler_HandleDmaBlock(1U, g_samples_per_channel);
    }
}
