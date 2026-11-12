// Package config loads settings from the environment
package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

type Config struct {
	PublicOrigin string
	HTTPAddr     string
	DatabaseURL  string
	S3           S3
}

type S3 struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
}

func Load(getenv func(string) string) (Config, error) {
	var errs []error
	required := func(key string) string {
		v := strings.TrimSpace(getenv(key))
		if v == "" {
			errs = append(errs, fmt.Errorf("%s: required", key))
		}
		return v
	}

	var cfg Config

	if origin := required("PUBLIC_ORIGIN"); origin != "" {
		normalized, err := parseOrigin(origin)
		if err != nil {
			errs = append(errs, fmt.Errorf("PUBLIC_ORIGIN: %w", err))
		}
		cfg.PublicOrigin = normalized
	}

	cfg.HTTPAddr = strings.TrimSpace(getenv("HTTP_ADDR"))
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	} else if _, _, err := net.SplitHostPort(cfg.HTTPAddr); err != nil {
		errs = append(errs, fmt.Errorf("HTTP_ADDR: %w", err))
	}

	cfg.DatabaseURL = required("DATABASE_URL")

	cfg.S3.Endpoint = required("S3_ENDPOINT")
	if cfg.S3.Endpoint != "" {
		if err := checkHTTPURL(cfg.S3.Endpoint); err != nil {
			errs = append(errs, fmt.Errorf("S3_ENDPOINT: %w", err))
		}
	}
	cfg.S3.Region = required("S3_REGION")
	cfg.S3.Bucket = required("S3_BUCKET")
	cfg.S3.AccessKey = required("S3_ACCESS_KEY")
	cfg.S3.SecretKey = required("S3_SECRET_KEY")

	if len(errs) > 0 {
		return Config{}, fmt.Errorf("invalid config: %w", errors.Join(errs...))
	}
	return cfg, nil
}

func parseOrigin(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if err := checkHTTP(u); err != nil {
		return "", err
	}
	if u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("must be an origin without path, query or credentials")
	}
	return strings.ToLower(u.Scheme + "://" + u.Host), nil
}

func checkHTTPURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	return checkHTTP(u)
}

func checkHTTP(u *url.URL) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("must use http or https")
	}
	if u.Host == "" {
		return errors.New("must include a host")
	}
	return nil
}
