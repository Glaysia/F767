#pragma once

#ifndef __cplusplus
#error "adc.hh requires a C++ translation unit"
#endif

#include <stdint.h>
#include <stddef.h>

enum
{
    kAdcSampleChannels = 2
};

struct AdcHandler
{
    static void Init(uint16_t *adc1_buffer, uint16_t *adc3_buffer, size_t samples_per_half);
    static void StartDma(void);
    static void Process(void);
};
