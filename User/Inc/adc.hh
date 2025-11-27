#pragma once

#ifndef __cplusplus
#error "adc.hh requires a C++ translation unit"
#endif

#include <stdint.h>

enum
{
    kAdcSampleChannels = 3
};

struct AdcSampleTriple
{
    uint16_t values[kAdcSampleChannels];
};



void AdcInit(void);
struct AdcSampleTriple AdcGetLatest(void);

