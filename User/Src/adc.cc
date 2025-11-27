#include "adc.hh"

static struct AdcSampleTriple g_latest_samples;

void AdcInit(void)
{
    for (int i = 0; i < kAdcSampleChannels; ++i)
    {
        g_latest_samples.values[i] = 0U;
    }
}

struct AdcSampleTriple AdcGetLatest(void)
{
    return g_latest_samples;
}

