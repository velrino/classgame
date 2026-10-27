package app

import (
	"classgame-api/internal/config"

	"github.com/labstack/echo/v5"
)

// ClientIP trusts only explicitly named proxy networks, in every environment.
func ClientIP(cidrs string) (echo.IPExtractor, error) {
	networks, err := config.ProxyNetworks(cidrs)
	if err != nil {
		return nil, err
	}
	if len(networks) == 0 {
		return echo.ExtractIPDirect(), nil
	}
	options := []echo.TrustOption{echo.TrustLoopback(false), echo.TrustPrivateNet(false), echo.TrustLinkLocal(false)}
	for _, network := range networks {
		options = append(options, echo.TrustIPRange(network))
	}
	return echo.ExtractIPFromXFFHeader(options...), nil
}
