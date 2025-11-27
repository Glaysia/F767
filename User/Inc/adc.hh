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

struct AdcSampleTriple
{
    uint16_t values[kAdcSampleChannels];

    static void Init(uint16_t *dma_buffer, size_t dma_samples);
    static void StartDma(void);
    static AdcSampleTriple GetLatest(void);
    static AdcSampleTriple &Instance(void);
};
