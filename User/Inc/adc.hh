#pragma once

#include <stdint.h>

enum
{
    kAdcSampleChannels = 3
};

struct AdcSampleTriple
{
    uint16_t values[kAdcSampleChannels];
};

#ifdef __cplusplus
extern "C" {
#endif

void AdcInit(void);
struct AdcSampleTriple AdcGetLatest(void);

#ifdef __cplusplus
}
#endif

