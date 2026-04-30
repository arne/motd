package builtin

import (
	"context"
	"net"
	"strings"

	"github.com/arne/motd/internal/block"
	"github.com/arne/motd/internal/module"
)

func ipAddrs(_ context.Context, _ module.Spec) (block.Block, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return block.Block{}, err
	}
	var lines []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok || ipnet.IP.To4() == nil {
				continue
			}
			lines = append(lines, ipnet.IP.String()+"  ("+categorizeIface(iface.Name)+")")
		}
	}
	return block.Block{Text: strings.Join(lines, "\n")}, nil
}

func categorizeIface(name string) string {
	switch {
	case strings.HasPrefix(name, "tailscale"):
		return "tailscale"
	case strings.HasPrefix(name, "incusbr"), strings.HasPrefix(name, "lxcbr"), strings.HasPrefix(name, "lxdbr"):
		return "incus"
	case strings.HasPrefix(name, "docker"), strings.HasPrefix(name, "br-"):
		return "docker"
	case strings.HasPrefix(name, "podman"), strings.HasPrefix(name, "cni-"):
		return "podman"
	case strings.HasPrefix(name, "virbr"):
		return "libvirt"
	case strings.HasPrefix(name, "wg"):
		return "wg"
	case strings.HasPrefix(name, "tun"), strings.HasPrefix(name, "tap"), strings.HasPrefix(name, "ppp"):
		return "vpn"
	case strings.HasPrefix(name, "zt"):
		return "zerotier"
	case strings.HasPrefix(name, "bond"):
		return "bond"
	case strings.HasPrefix(name, "vlan"):
		return "vlan"
	case strings.HasPrefix(name, "wlan"), strings.HasPrefix(name, "wlp"),
		strings.HasPrefix(name, "wlx"), strings.HasPrefix(name, "ath"):
		return "wifi"
	case strings.HasPrefix(name, "eth"), strings.HasPrefix(name, "enp"),
		strings.HasPrefix(name, "eno"), strings.HasPrefix(name, "ens"),
		strings.HasPrefix(name, "enx"):
		return "eth"
	}
	return name
}
