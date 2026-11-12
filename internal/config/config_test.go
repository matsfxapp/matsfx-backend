package config

import (
	"strings"
	"testing"
)

func validEnv() map[string]string {
	return map[string]string{
		"PUBLIC_ORIGIN": "https://matsfx.example.com",
		"DATABASE_URL":  "postgres://matsfx:secret@localhost:5432/matsfx",
		"S3_ENDPOINT":   "http://localhost:3900",
		"S3_REGION":     "garage",
		"S3_BUCKET":     "matsfx",
		"S3_ACCESS_KEY": "GKtest",
		"S3_SECRET_KEY": "secret",
	}
}

func load(env map[string]string) (Config, error) {
	return Load(func(key string) string { return env[key] })
}

func TestLoadValid(t *testing.T) {
	cfg, err := load(validEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := Config{
		PublicOrigin: "https://matsfx.example.com",
		HTTPAddr:     ":8080",
		DatabaseURL:  "postgres://matsfx:secret@localhost:5432/matsfx",
		S3: S3{
			Endpoint:  "http://localhost:3900",
			Region:    "garage",
			Bucket:    "matsfx",
			AccessKey: "GKtest",
			SecretKey: "secret",
		},
	}
	if cfg != want {
		t.Fatalf("got %+v, want %+v", cfg, want)
	}
}

func TestLoadReportsEveryMissingKey(t *testing.T) {
	_, err := load(map[string]string{})
	if err == nil {
		t.Fatalf("expected error")
	}
	for key := range validEnv() {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error does not mention %s: %v", key, err)
		}
	}
}

func TestLoadPublicOrigin(t *testing.T) {
	tests := []struct {
		value   string
		want    string
		wantErr bool
	}{
		{value: "https://matsfx.example.com", want: "https://matsfx.example.com"},
		{value: "https://matsfx.example.com/", want: "https://matsfx.example.com"},
		{value: "https://matSFX.Example.com", want: "https://matsfx.example.com"},
		{value: "http://localhost:5173", want: "http://localhost:5173"},
		{value: "https://matsfx.example.com/app", wantErr: true},
		{value: "https://matsfx.example.com?x=1", wantErr: true},
		{value: "https://user@matsfx.example.com", wantErr: true},
		{value: "ftp://matsfx.example.com", wantErr: true},
		{value: "matsfx.example.com", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			env := validEnv()
			env["PUBLIC_ORIGIN"] = tt.value
			cfg, err := load(env)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "PUBLIC_ORIGIN") {
					t.Fatalf("expected PUBLIC_ORIGIN error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.PublicOrigin != tt.want {
				t.Fatalf("got %q, want %q", cfg.PublicOrigin, tt.want)
			}
		})
	}
}

func TestLoadHTTPAddr(t *testing.T) {
	tests := []struct {
		value   string
		want    string
		wantErr bool
	}{
		{value: "", want: ":8080"},
		{value: "127.0.0.1:9000", want: "127.0.0.1:9000"},
		{value: "8080", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			env := validEnv()
			env["HTTP_ADDR"] = tt.value
			cfg, err := load(env)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "HTTP_ADDR") {
					t.Fatalf("expected HTTP_ADDR error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.HTTPAddr != tt.want {
				t.Fatalf("got %q, want %q", cfg.HTTPAddr, tt.want)
			}
		})
	}
}

func TestLoadS3Endpoint(t *testing.T) {
	for _, value := range []string{"localhost:3900", "s3://garage:3900"} {
		t.Run(value, func(t *testing.T) {
			env := validEnv()
			env["S3_ENDPOINT"] = value
			if _, err := load(env); err == nil || !strings.Contains(err.Error(), "S3_ENDPOINT") {
				t.Fatalf("expected S3_ENDPOINT error, got %v", err)
			}
		})
	}
}
