//go:build linux || darwin

package capture

import (
	"log"
	"net"
	"net/netip"
)

// ignoreSelfIPs 把网卡自身的单播 IP 登记进忽略集(单臂 NAT 去重,见 RunLive)。
func ignoreSelfIPs(e *Engine, iface string) {
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return
	}
	addrs, err := ifi.Addrs()
	if err != nil {
		return
	}
	var ips []netip.Addr
	for _, a := range addrs {
		var raw net.IP
		switch v := a.(type) {
		case *net.IPNet:
			raw = v.IP
		case *net.IPAddr:
			raw = v.IP
		}
		if ip, ok := netip.AddrFromSlice(raw); ok && !ip.IsLoopback() {
			ip = ip.Unmap()
			e.AddSkipIP(ip)
			ips = append(ips, ip)
		}
	}
	if len(ips) > 0 {
		log.Printf("单臂网关去重: 忽略本机 %s 的 IP %v", iface, ips)
	}
}
