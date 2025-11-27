#include "eth.hh"

enum
{
    kEthStreamSamplesPerFrame = kEthStreamChannels
};

static EthStream g_eth_stream;

EthStream &EthStream::Instance(void)
{
    return g_eth_stream;
}

void EthStream::Reset(void)
{
    packet_sequence = 0U;
    queued_frames = 0U;

    for (size_t i = 0; i < kEthStreamFrameCapacity * kEthStreamSamplesPerFrame; ++i)
    {
        frame_buffer[i] = 0U;
    }
}

void EthStream::QueueSamples(const uint16_t samples[kEthStreamChannels])
{
    if (samples == NULL)
    {
        return;
    }

    if (queued_frames >= kEthStreamFrameCapacity)
    {
        Flush();
    }

    size_t base_index = queued_frames * kEthStreamSamplesPerFrame;
    for (size_t i = 0; i < kEthStreamSamplesPerFrame; ++i)
    {
        frame_buffer[base_index + i] = samples[i];
    }

    ++queued_frames;
}

size_t EthStream::BytesReady(void) const
{
    return queued_frames * kEthStreamSamplesPerFrame * sizeof(uint16_t);
}

void EthStream::Flush(void)
{
    ++packet_sequence;
    queued_frames = 0U;
}
