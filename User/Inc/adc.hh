#pragma once

#ifndef __cplusplus
#error "adc.hh requires a C++ translation unit"
#endif

#include <stdint.h>
#include <stddef.h>

enum
{
    kAdcSampleChannels = 3
};

struct AdcHandler
{
    static void Init(uint16_t *dma_buffer, size_t dma_samples);
    static void StartDma(void);
    static void Process(void);
};
