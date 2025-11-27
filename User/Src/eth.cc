#include "eth.hh"

enum
{
    kEthStreamSamplesPerFrame = kEthStreamChannels
};

static struct EthStream g_eth_stream;

static void EthStreamFlush(struct EthStream *stream)
{
    if (stream == NULL)
    {
        return;
    }

    ++stream->packet_sequence;
    stream->queued_frames = 0;
}

struct EthStream *EthStreamGet(void)
{
    return &g_eth_stream;
}

void EthStreamReset(struct EthStream *stream)
{
    if (stream == NULL)
    {
        return;
    }

    stream->packet_sequence = 0U;
    stream->queued_frames = 0U;

    for (size_t i = 0; i < kEthStreamFrameCapacity * kEthStreamSamplesPerFrame; ++i)
    {
        stream->frame_buffer[i] = 0U;
    }
}

void EthStreamQueueSamples(struct EthStream *stream, const uint16_t samples[kEthStreamChannels])
{
    if ((stream == NULL) || (samples == NULL))
    {
        return;
    }

    if (stream->queued_frames >= kEthStreamFrameCapacity)
    {
        EthStreamFlush(stream);
    }

    size_t base_index = stream->queued_frames * kEthStreamSamplesPerFrame;
    for (size_t i = 0; i < kEthStreamSamplesPerFrame; ++i)
    {
        stream->frame_buffer[base_index + i] = samples[i];
    }

    ++stream->queued_frames;
}

size_t EthStreamBytesReady(const struct EthStream *stream)
{
    if (stream == NULL)
    {
        return 0U;
    }

    return stream->queued_frames * kEthStreamSamplesPerFrame * sizeof(uint16_t);
}

