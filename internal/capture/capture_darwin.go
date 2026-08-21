//go:build darwin

package capture

import (
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

// RunLive 在指定网卡上用 libpcap 实时抓包。阻塞运行。
func (e *Engine) RunLive(iface string) error {
	if !e.NoSelfIgnore {
		// 网关旁路去重:忽略网卡自身 IP,避免把 NAT 后的本机副本重复解析。
		ignoreSelfIPs(e, iface)
	}

	handle, err := pcap.OpenLive(iface, 65535, true, time.Second)
	if err != nil {
		return err
	}
	defer handle.Close()
	// 按链路类型解码,兼容以太网与 Wi-Fi(radiotap/802.11)。
	src := gopacket.NewPacketSource(handle, handle.LinkType().LayerType())
	src.NoCopy = true
	e.process(src)
	return nil
}
