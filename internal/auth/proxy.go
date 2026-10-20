package auth

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ConfigureTrustedProxies must be called before serving requests. No forwarded
// headers are trusted by default, including from private or loopback addresses.
func (s *Service) ConfigureTrustedProxies(config string) error {
	var networks []netip.Prefix
	for _, value := range strings.Split(config, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(value)
		if err != nil || prefix.Bits() == 0 {
			return fmt.Errorf("ungültiger oder unbegrenzter Proxy-CIDR %q", value)
		}
		networks = append(networks, prefix.Masked())
	}
	s.trustedProxies = networks
	return nil
}

func peerAddress(r *http.Request) netip.Addr {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip, _ := netip.ParseAddr(host)
	return ip.Unmap()
}

func (s *Service) trustedAddress(ip netip.Addr) bool {
	for _, prefix := range s.trustedProxies {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

func (s *Service) IsTrustedProxy(r *http.Request) bool {
	return s != nil && s.trustedAddress(peerAddress(r))
}

func (s *Service) clientIP(r *http.Request) string {
	peer := peerAddress(r)
	if !peer.IsValid() {
		return r.RemoteAddr
	}
	if !s.trustedAddress(peer) {
		return peer.String()
	}
	chain := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	if len(chain) > 32 {
		return peer.String()
	}
	current := peer
	for i := len(chain) - 1; i >= 0; i-- {
		if !s.trustedAddress(current) {
			return current.String()
		}
		next, err := netip.ParseAddr(strings.TrimSpace(chain[i]))
		if err != nil {
			return peer.String()
		}
		current = next.Unmap()
	}
	return current.String()
}
