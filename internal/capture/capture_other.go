//go:build !linux

package capture

import "errors"

// RunLive 实时抓包依赖 Linux 的 AF_PACKET,其他平台仅支持 -pcap 离线回放。
func (e *Engine) RunLive(iface string) error {
	return errors.New("实时抓包仅支持 Linux(AF_PACKET);当前平台请使用 -pcap 离线回放")
}