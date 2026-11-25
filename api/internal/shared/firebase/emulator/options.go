// Package emulator validates explicit, local-only Firebase emulator settings.
package emulator

import (
	"errors"
	"net"
	"regexp"
	"strconv"
	"strings"
)

type Options struct {
	Enabled                                         bool
	Environment, ProjectID, AuthHost, FirestoreHost string
	CredentialsConfigured                           bool
}

func (options Options) Validate() error {
	configured := options.AuthHost != "" || options.FirestoreHost != ""
	if !options.Enabled {
		if configured {
			return errors.New("Firebase emulator hosts require FIREBASE_EMULATORS_ENABLED=true")
		}
		return nil
	}
	if options.Environment != "development" || !regexp.MustCompile(`^demo-[a-z0-9][a-z0-9-]{0,50}$`).MatchString(options.ProjectID) || options.CredentialsConfigured {
		return errors.New("Firebase emulators require development, a demo- project, and no credentials")
	}
	if !loopback(options.AuthHost) || !loopback(options.FirestoreHost) {
		return errors.New("Firebase emulators require both Auth and Firestore numeric loopback host:port settings")
	}
	return nil
}

func loopback(value string) bool {
	host, port, err := net.SplitHostPort(strings.TrimSpace(value))
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return false
	}
	n, err := strconv.Atoi(port)
	return err == nil && n > 0 && n <= 65535
}
