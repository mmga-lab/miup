package playground

import (
	"testing"
)

func TestModeConstant(t *testing.T) {
	if ModeStandalone != "standalone" {
		t.Errorf("ModeStandalone = %s, want standalone", ModeStandalone)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Tag != "default" {
		t.Errorf("Tag = %s, want default", cfg.Tag)
	}
	if cfg.Mode != ModeStandalone {
		t.Errorf("Mode = %s, want standalone", cfg.Mode)
	}
	if cfg.MilvusVersion == "" {
		t.Error("MilvusVersion should not be empty")
	}
	if cfg.EtcdVersion == "" {
		t.Error("EtcdVersion should not be empty")
	}
	if cfg.MinioVersion == "" {
		t.Error("MinioVersion should not be empty")
	}
	if cfg.WithMonitor != false {
		t.Error("WithMonitor should be false by default")
	}
	if cfg.MilvusPort != 19530 {
		t.Errorf("MilvusPort = %d, want 19530", cfg.MilvusPort)
	}
	if cfg.EtcdPort != 2379 {
		t.Errorf("EtcdPort = %d, want 2379", cfg.EtcdPort)
	}
	if cfg.MinioPort != 9000 {
		t.Errorf("MinioPort = %d, want 9000", cfg.MinioPort)
	}
	if cfg.MinioConsole != 9001 {
		t.Errorf("MinioConsole = %d, want 9001", cfg.MinioConsole)
	}
	if cfg.PrometheusPort != 9090 {
		t.Errorf("PrometheusPort = %d, want 9090", cfg.PrometheusPort)
	}
	if cfg.GrafanaPort != 3000 {
		t.Errorf("GrafanaPort = %d, want 3000", cfg.GrafanaPort)
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Run("empty config gets defaults", func(t *testing.T) {
		cfg := &Config{}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Validate() error = %v", err)
		}
		if cfg.Tag != "default" {
			t.Errorf("Tag = %s, want default", cfg.Tag)
		}
		if cfg.Mode != ModeStandalone {
			t.Errorf("Mode = %s, want standalone", cfg.Mode)
		}
		if cfg.MilvusVersion != "v2.5.4" {
			t.Errorf("MilvusVersion = %s, want v2.5.4", cfg.MilvusVersion)
		}
	})

	t.Run("non-empty values preserved", func(t *testing.T) {
		cfg := &Config{
			Tag:           "custom",
			Mode:          ModeStandalone,
			MilvusVersion: "v2.4.0",
		}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Validate() error = %v", err)
		}
		if cfg.Tag != "custom" {
			t.Errorf("Tag = %s, want custom", cfg.Tag)
		}
		if cfg.MilvusVersion != "v2.4.0" {
			t.Errorf("MilvusVersion = %s, want v2.4.0", cfg.MilvusVersion)
		}
	})
}

func TestConfigStruct(t *testing.T) {
	cfg := Config{
		Tag:            "test",
		Mode:           ModeStandalone,
		MilvusVersion:  "v2.4.0",
		EtcdVersion:    "3.5.0",
		MinioVersion:   "latest",
		WithMonitor:    true,
		MilvusPort:     19530,
		EtcdPort:       2379,
		MinioPort:      9000,
		MinioConsole:   9001,
		PrometheusPort: 9090,
		GrafanaPort:    3000,
	}

	if cfg.Tag != "test" {
		t.Errorf("Tag = %s, want test", cfg.Tag)
	}
	if cfg.Mode != ModeStandalone {
		t.Errorf("Mode = %s, want standalone", cfg.Mode)
	}
	if cfg.WithMonitor != true {
		t.Error("WithMonitor should be true")
	}
	if cfg.MilvusVersion != "v2.4.0" {
		t.Errorf("MilvusVersion = %s, want v2.4.0", cfg.MilvusVersion)
	}
	if cfg.EtcdVersion != "3.5.0" {
		t.Errorf("EtcdVersion = %s, want 3.5.0", cfg.EtcdVersion)
	}
	if cfg.MinioVersion != "latest" {
		t.Errorf("MinioVersion = %s, want latest", cfg.MinioVersion)
	}
	if cfg.MilvusPort != 19530 {
		t.Errorf("MilvusPort = %d, want 19530", cfg.MilvusPort)
	}
	if cfg.EtcdPort != 2379 {
		t.Errorf("EtcdPort = %d, want 2379", cfg.EtcdPort)
	}
	if cfg.MinioPort != 9000 {
		t.Errorf("MinioPort = %d, want 9000", cfg.MinioPort)
	}
	if cfg.MinioConsole != 9001 {
		t.Errorf("MinioConsole = %d, want 9001", cfg.MinioConsole)
	}
	if cfg.PrometheusPort != 9090 {
		t.Errorf("PrometheusPort = %d, want 9090", cfg.PrometheusPort)
	}
	if cfg.GrafanaPort != 3000 {
		t.Errorf("GrafanaPort = %d, want 3000", cfg.GrafanaPort)
	}
}
