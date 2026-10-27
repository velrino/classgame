package uploadcare

import (
	"errors"
	"net/url"
	"strings"
)

var (
	ErrInvalidCDNHost = errors.New("uploadcare cdn url host is not allowed")
	ErrInvalidCDNURL  = errors.New("uploadcare cdn url must be a valid https url")
)

var cdnHostSuffixes = [...]string{
	"ucarecdn.com",
	"ucarecd.net",
}

func ValidateCDNURL(parsedURL *url.URL) error {
	if parsedURL == nil || parsedURL.Scheme != "https" || parsedURL.User != nil {
		return ErrInvalidCDNURL
	}
	if port := parsedURL.Port(); port != "" && port != "443" {
		return ErrInvalidCDNURL
	}
	if !IsCDNHost(parsedURL.Hostname()) {
		return ErrInvalidCDNHost
	}

	return nil
}

func IsCDNHost(host string) bool {
	normalizedHost := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	if normalizedHost == "" {
		return false
	}

	for _, suffix := range cdnHostSuffixes {
		if normalizedHost == suffix || strings.HasSuffix(normalizedHost, "."+suffix) {
			return true
		}
	}

	return false
}
