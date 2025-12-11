#include "adc.hh"

#include "eth.hh"

extern "C" {
#include "stm32f7xx_hal.h"

extern ADC_HandleTypeDef hadc1;
extern TIM_HandleTypeDef htim5;
void Error_Handler(void);
}

enum
{
    kAdcFrameSamples = kEthStreamFrameCapacity * kEthStreamChannels,
    kAdcFrameQueueDepth = 512
};

struct AdcFrame
{
    uint8_t samples[kAdcFrameSamples];
    size_t sample_count;
    uint64_t first_sample_idx;
    uint16_t flags;
};

static uint16_t *g_adc_dma_buffer = NULL;
static size_t g_adc_dma_samples = 0U;
static size_t g_half_samples = 0U;
static uint64_t g_next_sample_idx = 0U;

static AdcFrame g_frame_queue[kAdcFrameQueueDepth];
static volatile size_t g_frame_read = 0U;
static volatile size_t g_frame_write = 0U;
static volatile uint16_t g_drop_latch = 0U;

static void AdcHandler_HandleDmaBlock(size_t base_index);
static void AdcHandler_Enqueue(const uint16_t *src, size_t samples);

void AdcHandler::Init(uint16_t *dma_buffer, size_t dma_samples)
{
    g_adc_dma_buffer = dma_buffer;
    g_adc_dma_samples = dma_samples;

    if ((g_adc_dma_buffer == NULL) || (g_adc_dma_samples == 0U) || ((g_adc_dma_samples % 2U) != 0U))
    {
        Error_Handler();
    }

    g_half_samples = g_adc_dma_samples / 2U;
    if (g_half_samples != kAdcFrameSamples)
    {
        Error_Handler();
    }

    g_frame_read = 0U;
    g_frame_write = 0U;
    g_drop_latch = 0U;
    g_next_sample_idx = 0U;
}

void AdcHandler::StartDma(void)
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

static void AdcHandler_HandleDmaBlock(size_t base_index)
{
    if ((g_adc_dma_buffer == NULL) || (g_half_samples == 0U))
    {
        return;
    }

    if ((base_index + g_half_samples) > g_adc_dma_samples)
    {
        return;
    }

    AdcHandler_Enqueue(&g_adc_dma_buffer[base_index], g_half_samples);
}

static void AdcHandler_Enqueue(const uint16_t *src, size_t samples)
{
    if (samples > kAdcFrameSamples)
    {
        samples = kAdcFrameSamples;
    }

    const size_t samples_per_ch = (kEthStreamChannels == 0U) ? 0U : (samples / kEthStreamChannels);
    const uint64_t first_idx = g_next_sample_idx;
    g_next_sample_idx += (uint64_t)samples_per_ch;

    size_t next_write = (g_frame_write + 1U) % kAdcFrameQueueDepth;
    if ((next_write == g_frame_read) || (samples_per_ch == 0U))
    {
        g_drop_latch = 1U;
        return;
    }

    AdcFrame &frame = g_frame_queue[g_frame_write];
    for (size_t i = 0; i < samples; ++i)
    {
        frame.samples[i] = (uint8_t)(src[i] & 0xFFU);
    }
    frame.sample_count = samples;
    frame.first_sample_idx = first_idx;
    frame.flags = g_drop_latch;
    g_drop_latch = 0U;

    g_frame_write = next_write;
}

extern "C" void HAL_ADC_ConvHalfCpltCallback(ADC_HandleTypeDef *hadc)
{
    if (hadc != &hadc1)
    {
        return;
    }

    AdcHandler_HandleDmaBlock(0U);
}

extern "C" void HAL_ADC_ConvCpltCallback(ADC_HandleTypeDef *hadc)
{
    if (hadc != &hadc1)
    {
        return;
    }

    AdcHandler_HandleDmaBlock(g_half_samples);
}
