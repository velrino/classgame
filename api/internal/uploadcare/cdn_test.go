package uploadcare

import (
	"net/url"
	"testing"
)

func TestIsCDNHostAcceptsUploadcareCDNHosts(t *testing.T) {
	t.Parallel()

	for _, host := range []string{
		"ucarecdn.com",
		"files.ucarecdn.com",
		"3n8i12a936.ucarecd.net",
		"ucarecd.net",
		"UCARECDN.COM.",
	} {
		if !IsCDNHost(host) {
			t.Fatalf("expected %q to be accepted", host)
		}
	}
}

func TestIsCDNHostRejectsNonUploadcareCDNHosts(t *testing.T) {
	t.Parallel()

	for _, host := range []string{
		"",
		"evilucarecdn.com",
		"ucarecdn.com.example.com",
		"example.com",
	} {
		if IsCDNHost(host) {
			t.Fatalf("expected %q to be rejected", host)
		}
	}
}

func TestValidateCDNURLAcceptsProjectCDNURL(t *testing.T) {
	t.Parallel()

	parsedURL, err := url.Parse("https://3n8i12a936.ucarecd.net/958e51c7-4da4-41e1-a0f0-ecf29928c645/")
	if err != nil {
		t.Fatalf("parse test url: %v", err)
	}

	if err := ValidateCDNURL(parsedURL); err != nil {
		t.Fatalf("expected project cdn url to be accepted: %v", err)
	}
}
