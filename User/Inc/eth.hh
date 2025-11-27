#pragma once

#ifndef __cplusplus
#error "eth.hh requires a C++ translation unit"
#endif

#include <stddef.h>
#include <stdint.h>

enum
{
    kEthStreamChannels = 1,
    kEthStreamFrameCapacity = 64,
    kEthStreamSampleBits = 8
};

struct EthPacketHeader
{
    uint32_t packet_seq;
    uint64_t first_sample_idx;
    uint16_t channels;
    uint16_t samples_per_ch;
    uint16_t flags;
    uint16_t sample_bits;
} __attribute__((packed));

struct udp_pcb;

struct EthStream
{
    uint32_t packet_sequence;
    uint64_t first_sample_index;
    struct udp_pcb *udp;

    void Reset(void);
    bool SendFrame(const uint16_t *samples, size_t sample_count, uint16_t flags);
    static EthStream &Instance(void);

private:
    EthStream(void) : packet_sequence(0U), first_sample_index(0U), udp(NULL) {}
};
