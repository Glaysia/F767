#pragma once

#ifndef __cplusplus
#error "eth.hh requires a C++ translation unit"
#endif

#include <stddef.h>
#include <stdint.h>

enum
{
    kEthStreamChannels = 3,
    kEthStreamFrameCapacity = 64
};

struct EthStream
{
    uint32_t packet_sequence;
    size_t queued_frames;
    uint16_t frame_buffer[kEthStreamFrameCapacity * kEthStreamChannels];

    void Reset(void);
    void QueueSamples(const uint16_t samples[kEthStreamChannels]);
    size_t BytesReady(void) const;
    static EthStream &Instance(void);

private:
    void Flush(void);
};
