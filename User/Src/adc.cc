#include "adc.hh"

#include "eth.hh"

extern "C" {
#include "stm32f7xx_hal.h"

extern ADC_HandleTypeDef hadc1;
extern ADC_HandleTypeDef hadc3;
extern TIM_HandleTypeDef htim5;
void Error_Handler(void);
}

enum
{
    kAdcFrameSamples = kEthStreamFrameCapacity * kEthStreamChannels,
    kAdcFrameQueueDepth = 512,
    kAdcMaskAdc1 = 0x1U,
    kAdcMaskAdc3 = 0x2U,
    kAdcMaskBoth = kAdcMaskAdc1 | kAdcMaskAdc3
};

struct AdcFrame
{
    uint8_t samples[kAdcFrameSamples];
    size_t sample_count;
    uint64_t first_sample_idx;
    uint16_t flags;
};

static uint16_t *g_adc1_dma_buffer = NULL;
static uint16_t *g_adc3_dma_buffer = NULL;
static size_t g_samples_per_half = 0U;
static size_t g_frame_samples = 0U;
static uint64_t g_next_sample_idx = 0U;

static AdcFrame g_frame_queue[kAdcFrameQueueDepth];
static volatile size_t g_frame_read = 0U;
static volatile size_t g_frame_write = 0U;
static volatile uint16_t g_drop_latch = 0U;
static volatile uint8_t g_ready_mask[2] = {0U, 0U}; // bit0=ADC1, bit1=ADC3 for each DMA half

static void AdcHandler_MarkReady(uint8_t adc_mask, size_t half_index);
static void AdcHandler_Enqueue(size_t base_index);

void AdcHandler::Init(uint16_t *adc1_buffer, uint16_t *adc3_buffer, size_t samples_per_half)
{
    g_adc1_dma_buffer = adc1_buffer;
    g_adc3_dma_buffer = adc3_buffer;
    g_samples_per_half = samples_per_half;
    g_frame_samples = g_samples_per_half * kEthStreamChannels;

    if ((g_adc1_dma_buffer == NULL) || (g_adc3_dma_buffer == NULL) || (g_samples_per_half == 0U))
    {
        Error_Handler();
    }

    if (g_frame_samples != kAdcFrameSamples)
    {
        Error_Handler();
    }

    g_frame_read = 0U;
    g_frame_write = 0U;
    g_drop_latch = 0U;
    g_next_sample_idx = 0U;
    g_ready_mask[0] = 0U;
    g_ready_mask[1] = 0U;
}

void AdcHandler::StartDma(void)
{
    const size_t dma_samples = g_samples_per_half * 2U;

    if ((g_adc1_dma_buffer == NULL) || (g_adc3_dma_buffer == NULL) || (dma_samples == 0U))
    {
        Error_Handler();
    }

    if (HAL_ADC_Start_DMA(&hadc1, (uint32_t *)g_adc1_dma_buffer, dma_samples) != HAL_OK)
    {
        Error_Handler();
    }
    if (HAL_ADC_Start_DMA(&hadc3, (uint32_t *)g_adc3_dma_buffer, dma_samples) != HAL_OK)
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
        const bool sent = EthStream::Instance().SendFrame(frame.samples, frame.sample_count, frame.flags, frame.first_sample_idx);
        if (!sent)
        {
            g_drop_latch = 1U;
        }

        g_frame_read = (g_frame_read + 1U) % kAdcFrameQueueDepth;
    }
}

static void AdcHandler_MarkReady(uint8_t adc_mask, size_t half_index)
{
    if (half_index >= 2U)
    {
        return;
    }

    const uint8_t prev = g_ready_mask[half_index];
    g_ready_mask[half_index] |= adc_mask;
    if ((prev & adc_mask) != 0U)
    {
        g_drop_latch = 1U;
    }

    if (g_ready_mask[half_index] == kAdcMaskBoth)
    {
        g_ready_mask[half_index] = 0U;
        const size_t base_index = half_index * g_samples_per_half;
        AdcHandler_Enqueue(base_index);
    }
}

static void AdcHandler_Enqueue(size_t base_index)
{
    if ((g_adc1_dma_buffer == NULL) || (g_adc3_dma_buffer == NULL) || (g_frame_samples == 0U))
    {
        return;
    }

    const size_t dma_samples = g_samples_per_half * 2U;
    if ((base_index + g_samples_per_half) > dma_samples)
    {
        g_drop_latch = 1U;
        return;
    }

    const uint16_t *src[2] = {&g_adc1_dma_buffer[base_index], &g_adc3_dma_buffer[base_index]};
    const size_t samples = g_frame_samples;
    const size_t samples_per_ch = g_samples_per_half;

    size_t next_write = (g_frame_write + 1U) % kAdcFrameQueueDepth;
    if ((next_write == g_frame_read) || (samples_per_ch == 0U))
    {
        g_drop_latch = 1U;
        return;
    }

    const uint64_t first_idx = g_next_sample_idx;
    g_next_sample_idx += (uint64_t)samples_per_ch;

    AdcFrame &frame = g_frame_queue[g_frame_write];
    for (size_t i = 0; i < samples_per_ch; ++i)
    {
        const size_t base = i * kEthStreamChannels;
        frame.samples[base + 0U] = (uint8_t)(src[0][i] & 0xFFU);
        frame.samples[base + 1U] = (uint8_t)(src[1][i] & 0xFFU);
    }
    frame.sample_count = samples;
    frame.first_sample_idx = first_idx;
    frame.flags = g_drop_latch;
    g_drop_latch = 0U;

    g_frame_write = next_write;
}

extern "C" void HAL_ADC_ConvHalfCpltCallback(ADC_HandleTypeDef *hadc)
{
    if (hadc == &hadc1)
    {
        AdcHandler_MarkReady(kAdcMaskAdc1, 0U);
    }
    else if (hadc == &hadc3)
    {
        AdcHandler_MarkReady(kAdcMaskAdc3, 0U);
    }
}

extern "C" void HAL_ADC_ConvCpltCallback(ADC_HandleTypeDef *hadc)
{
    if (hadc == &hadc1)
    {
        AdcHandler_MarkReady(kAdcMaskAdc1, 1U);
    }
    else if (hadc == &hadc3)
    {
        AdcHandler_MarkReady(kAdcMaskAdc3, 1U);
    }
}
