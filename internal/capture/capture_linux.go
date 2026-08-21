//go:build linux

package capture

import (
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/afpacket"
	"github.com/google/gopacket/layers"
)

// RunLive 在指定网卡上用 AF_PACKET 被动抓包(无需 libpcap)。阻塞运行。
func (e *Engine) RunLive(iface string) error {
	if !e.NoSelfIgnore {
		// 单臂网关去重:抓包网卡在做 SNAT 转发时,会把游戏流的一个副本(源改为本机 IP)
		// 再次从同一网卡发出并被捕获。登记本机 IP 到忽略集,只保留 NAT 前的真实客户端会话。
		ignoreSelfIPs(e, iface)
	}

	tp, err := afpacket.NewTPacket(
		afpacket.OptInterface(iface),
		afpacket.OptPollTimeout(time.Second),
	)
	if err != nil {
		return err
	}
	defer tp.Close()
	src := gopacket.NewPacketSource(tp, layers.LayerTypeEthernet)
	src.NoCopy = true
	e.process(src)
	return nil
}
