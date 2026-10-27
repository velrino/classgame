package config

import (
	"fmt"
	"net"
	"strings"

	"classgame-api/internal/auth"
)

func ProxyNetworks(value string) ([]*net.IPNet, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	if len(parts) > 32 {
		return nil, fmt.Errorf("at most 32 trusted proxy networks are allowed")
	}
	networks := make([]*net.IPNet, 0, len(parts))
	for _, part := range parts {
		_, network, err := net.ParseCIDR(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("trusted proxies must be CIDR networks")
		}
		ones, _ := network.Mask.Size()
		if ones == 0 {
			return nil, fmt.Errorf("trusted proxies cannot include every address")
		}
		networks = append(networks, network)
	}
	return networks, nil
}

func validateProxyNetworks(value any) error { _, err := ProxyNetworks(value.(string)); return err }
func validateTeacherEmails(environment string) func(any) error {
	return func(value any) error {
		emails := value.(string)
		if environment == "production" && strings.TrimSpace(emails) == "" {
			return fmt.Errorf("production requires an explicit teacher email allowlist")
		}
		_, err := auth.ParseTeacherEmails(emails)
		return err
	}
}
