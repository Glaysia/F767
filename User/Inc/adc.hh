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
    static void Init(uint16_t *adc1_buffer, uint16_t *adc2_buffer, size_t samples_per_adc);
    static void StartDma(void);
    static void Process(void);
};
