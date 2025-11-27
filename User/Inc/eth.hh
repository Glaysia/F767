#pragma once

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
};

#ifdef __cplusplus
extern "C" {
#endif

struct EthStream *EthStreamGet(void);
void EthStreamReset(struct EthStream *stream);
void EthStreamQueueSamples(struct EthStream *stream, const uint16_t samples[kEthStreamChannels]);
size_t EthStreamBytesReady(const struct EthStream *stream);

#ifdef __cplusplus
}
#endif

