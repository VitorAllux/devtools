package ssh

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/VitorAllux/devtools/internal/ui"
)

const (
	connectLoaderMinimum = 650 * time.Millisecond
	connectProbeTimeout  = 2500 * time.Millisecond
)

type sshEndpoint struct {
	Host string
	Port string
}

func (m *Manager) runConnectLoader(ctx context.Context, entry Entry) error {
	label := strings.TrimSpace(entry.Name)
	if label == "" {
		label = entry.Target
	}

	probe := m.probeConnection
	if m.connectionProbe != nil {
		probe = m.connectionProbe
	}

	return ui.RunWithRoyalLoader(ui.LoaderOptions{
		Action:  "connecting",
		Subject: label,
		Minimum: connectLoaderMinimum,
	}, func() error {
		probeCtx, cancel := context.WithTimeout(ctx, connectProbeTimeout)
		defer cancel()
		return probe(probeCtx, entry.Target)
	})
}

func (m *Manager) probeConnection(ctx context.Context, target string) error {
	endpoint, ok := m.resolveEndpoint(ctx, target)
	if !ok {
		return nil
	}

	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(endpoint.Host, endpoint.Port))
	if err != nil {
		return fmt.Errorf("cannot reach %s:%s", endpoint.Host, endpoint.Port)
	}
	return conn.Close()
}

func (m *Manager) resolveEndpoint(ctx context.Context, target string) (sshEndpoint, bool) {
	if _, err := m.Runner.LookPath("ssh"); err == nil {
		if output, err := m.Runner.Output(ctx, "", "ssh", "-G", target); err == nil {
			if endpoint, ok := parseSSHConfigEndpoint(string(output)); ok {
				return endpoint, true
			}
		}
	}
	return parseTargetEndpoint(target)
}

func parseSSHConfigEndpoint(output string) (sshEndpoint, bool) {
	endpoint := sshEndpoint{Port: "22"}
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch strings.ToLower(fields[0]) {
		case "hostname":
			endpoint.Host = fields[1]
		case "port":
			if isPort(fields[1]) {
				endpoint.Port = fields[1]
			}
		}
	}
	if endpoint.Host == "" || strings.Contains(endpoint.Host, "%") {
		return sshEndpoint{}, false
	}
	return endpoint, true
}

func parseTargetEndpoint(target string) (sshEndpoint, bool) {
	host := strings.TrimSpace(target)
	if host == "" {
		return sshEndpoint{}, false
	}
	if index := strings.LastIndex(host, "@"); index >= 0 {
		host = host[index+1:]
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return sshEndpoint{}, false
	}

	endpoint := sshEndpoint{Host: host, Port: "22"}
	if parsedHost, parsedPort, err := net.SplitHostPort(host); err == nil {
		endpoint.Host = strings.Trim(parsedHost, "[]")
		endpoint.Port = parsedPort
		return endpoint, endpoint.Host != "" && isPort(endpoint.Port)
	}

	if strings.Count(host, ":") == 1 {
		parsedHost, parsedPort, _ := strings.Cut(host, ":")
		if isPort(parsedPort) {
			endpoint.Host = strings.Trim(parsedHost, "[]")
			endpoint.Port = parsedPort
		}
	}
	return endpoint, endpoint.Host != "" && isPort(endpoint.Port)
}

func isPort(value string) bool {
	port, err := strconv.Atoi(value)
	return err == nil && port > 0 && port <= 65535
}
