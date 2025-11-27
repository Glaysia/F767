#include "eth.hh"

#include <string.h>

extern "C" {
#include "lwip/err.h"
#include "lwip/ip_addr.h"
#include "lwip/pbuf.h"
#include "lwip/udp.h"
}

enum
{
    kEthRemoteIp0 = 192,
    kEthRemoteIp1 = 168,
    kEthRemoteIp2 = 10,
    kEthRemoteIp3 = 1,
    kEthRemotePort = 5000
};

EthStream &EthStream::Instance(void)
{
    static EthStream instance;
    return instance;
}

void EthStream::Reset(void)
{
    packet_sequence = 0U;
    first_sample_index = 0U;

    if (udp != NULL)
    {
        udp_remove(udp);
        udp = NULL;
    }

    udp = udp_new();
    if (udp == NULL)
    {
        return;
    }

    ip_addr_t remote_ip;
    IP4_ADDR(&remote_ip, kEthRemoteIp0, kEthRemoteIp1, kEthRemoteIp2, kEthRemoteIp3);
    const err_t conn = udp_connect(udp, &remote_ip, kEthRemotePort);
    if (conn != ERR_OK)
    {
        udp_remove(udp);
        udp = NULL;
    }
}

bool EthStream::SendFrame(const uint16_t *samples, size_t sample_count, uint16_t flags)
{
    if ((samples == NULL) || (sample_count == 0U) || (udp == NULL))
    {
        return false;
    }

    if ((sample_count % kEthStreamChannels) != 0U)
    {
        return false;
    }

    const size_t sample_bytes = (kEthStreamSampleBits + 7U) / 8U;
    if (sample_bytes == 0U)
    {
        return false;
    }
    const uint16_t samples_per_ch = (uint16_t)(sample_count / kEthStreamChannels);
    const size_t payload_bytes = sample_count * sample_bytes;
    const size_t total_bytes = sizeof(EthPacketHeader) + payload_bytes;

    struct pbuf *p = pbuf_alloc(PBUF_TRANSPORT, (u16_t)total_bytes, PBUF_RAM);
    if (p == NULL)
    {
        return false;
    }

    if (p->len < total_bytes)
    {
        pbuf_free(p);
        return false;
    }

    EthPacketHeader *hdr = static_cast<EthPacketHeader *>(p->payload);
    hdr->packet_seq = packet_sequence++;
    hdr->first_sample_idx = first_sample_index;
    hdr->channels = kEthStreamChannels;
    hdr->samples_per_ch = samples_per_ch;
    hdr->flags = flags;
    hdr->sample_bits = kEthStreamSampleBits;

    first_sample_index += (uint64_t)samples_per_ch;

    uint8_t *payload = reinterpret_cast<uint8_t *>(p->payload) + sizeof(EthPacketHeader);
    if (sample_bytes == sizeof(uint16_t))
    {
        memcpy(payload, samples, payload_bytes);
    }
    else
    {
        for (size_t i = 0; i < sample_count; ++i)
        {
            payload[i] = (uint8_t)(samples[i] & 0xFFU);
        }
    }

    const err_t err = udp_send(udp, p);
    pbuf_free(p);

    return (err == ERR_OK);
}
