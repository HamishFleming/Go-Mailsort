package yahoo

import (
	"log"
	"os"
	"strconv"

	"github.com/HamishFleming/Go-Mailsort/internal/imapclient"
)

func NewProvider() (imapclient.Provider, error) {
	loadEnv(".env")
	return &Provider{}, nil
}

type Provider struct{}

func (p *Provider) Connect(cfg *imapclient.Config) (*imapclient.Client, error) {
	host := getEnv("IMAP_HOST")
	if host == "" {
		host = getEnv("YAHOO_IMAP_HOST")
	}
	user := getEnv("IMAP_USER")
	if user == "" {
		user = getEnv("YAHOO_EMAIL")
	}
	pass := getEnv("IMAP_PASS")
	if pass == "" {
		pass = getEnv("YAHOO_APP_PASSWORD")
	}
	portStr := getEnv("IMAP_PORT")
	if portStr == "" {
		portStr = getEnv("YAHOO_IMAP_PORT")
	}

	port := 993
	if n, err := strconv.Atoi(portStr); err == nil {
		port = n
	}

	cfg.Host = host
	cfg.User = user
	cfg.Pass = pass
	cfg.Port = port
	cfg.UseTLS = true

	log.Printf("[INFO] yahoo: connecting to %s:%d as %s", host, port, user)
	return imapclient.Connect(cfg)
}

func getEnv(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	log.Printf("[WARN] %s not set", key)
	return ""
}

func loadEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	for _, line := range splitLines(string(data)) {
		line = stripComment(line)
		if line == "" {
			continue
		}

		key, value, ok := splitKeyValue(line)
		if !ok {
			continue
		}

		os.Setenv(key, value)
		log.Printf("[DEBUG] .env: %s set", key)
	}
}

func splitLines(s string) []string {
	var lines []string
	prev := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[prev:i])
			prev = i + 1
		}
	}
	if prev < len(s) {
		lines = append(lines, s[prev:])
	}
	return lines
}

func stripComment(s string) string {
	for i, c := range s {
		if c == '#' && (i == 0 || s[i-1] == ' ') {
			return s[:i]
		}
	}
	return s
}

func splitKeyValue(s string) (string, string, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return s[:i], s[i+1:], true
		}
	}
	return "", "", false
}