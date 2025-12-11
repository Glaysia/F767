#include "fg_relay.h"

#include <string.h>

#include "main.h"

#include "stm32f7xx_hal.h"

#include "lwip/pbuf.h"
#include "lwip/udp.h"

enum
{
    kFgCtrlPort = 6001,
    kFgUartTimeoutMs = 50U,
    kFgMaxPayload = 128U
};

extern UART_HandleTypeDef huart4;

static struct udp_pcb *g_fg_pcb = NULL;

static void FgRelay_HandleUdp(void *arg, struct udp_pcb *pcb, struct pbuf *p, const ip_addr_t *addr, u16_t port)
{
    (void)arg;
    (void)pcb;
    (void)addr;
    (void)port;

    if (p == NULL)
    {
        return;
    }

    uint8_t buffer[kFgMaxPayload + 2U];
    uint16_t copy_len = (p->tot_len < kFgMaxPayload) ? (uint16_t)p->tot_len : kFgMaxPayload;
    if (copy_len > sizeof(buffer))
    {
        copy_len = (uint16_t)sizeof(buffer);
    }

    const uint16_t read = pbuf_copy_partial(p, buffer, copy_len, 0U);
    if (read == 0U)
    {
        pbuf_free(p);
        return;
    }

    uint16_t tx_len = read;
    if ((buffer[read - 1U] != '\n') && (buffer[read - 1U] != '\r'))
    {
        buffer[tx_len] = '\n';
        tx_len++;
    }

    (void)HAL_UART_Transmit(&huart4, buffer, tx_len, kFgUartTimeoutMs);

    pbuf_free(p);
}

void FgRelay_Init(void)
{
    if (g_fg_pcb != NULL)
    {
        udp_remove(g_fg_pcb);
        g_fg_pcb = NULL;
    }

    g_fg_pcb = udp_new();
    if (g_fg_pcb == NULL)
    {
        return;
    }

    const err_t bind = udp_bind(g_fg_pcb, IP_ADDR_ANY, kFgCtrlPort);
    if (bind != ERR_OK)
    {
        udp_remove(g_fg_pcb);
        g_fg_pcb = NULL;
        return;
    }

    udp_recv(g_fg_pcb, FgRelay_HandleUdp, NULL);
}

void FgRelay_Process(void)
{
    /* Placeholder for future async work; no-op for now. */
}
