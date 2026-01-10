package playground

import (
	"strings"
	"testing"
)

func TestGenerateComposeFile(t *testing.T) {
	t.Run("standalone without monitor", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.WithMonitor = false

		content, err := GenerateComposeFile(cfg)
		if err != nil {
			t.Fatalf("GenerateComposeFile() error = %v", err)
		}

		// Should contain required services
		if !strings.Contains(content, "etcd:") {
			t.Error("Should contain etcd service")
		}
		if !strings.Contains(content, "minio:") {
			t.Error("Should contain minio service")
		}
		if !strings.Contains(content, "standalone:") {
			t.Error("Should contain standalone service")
		}

		// Should NOT contain monitoring services
		if strings.Contains(content, "prometheus:") {
			t.Error("Should not contain prometheus service")
		}
		if strings.Contains(content, "grafana:") {
			t.Error("Should not contain grafana service")
		}

		// Check container names include tag
		if !strings.Contains(content, "milvus-etcd-default") {
			t.Error("Etcd container name should include tag")
		}
		if !strings.Contains(content, "milvus-minio-default") {
			t.Error("Minio container name should include tag")
		}
		if !strings.Contains(content, "milvus-standalone-default") {
			t.Error("Standalone container name should include tag")
		}

		// Check ports
		if !strings.Contains(content, "19530:19530") {
			t.Error("Should expose Milvus port")
		}
	})

	t.Run("standalone with monitor", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.WithMonitor = true

		content, err := GenerateComposeFile(cfg)
		if err != nil {
			t.Fatalf("GenerateComposeFile() error = %v", err)
		}

		// Should contain monitoring services
		if !strings.Contains(content, "prometheus:") {
			t.Error("Should contain prometheus service")
		}
		if !strings.Contains(content, "grafana:") {
			t.Error("Should contain grafana service")
		}

		// Check monitoring ports
		if !strings.Contains(content, "9090:9090") {
			t.Error("Should expose Prometheus port")
		}
		if !strings.Contains(content, "3000:3000") {
			t.Error("Should expose Grafana port")
		}

		// Check monitoring volumes
		if !strings.Contains(content, "prometheus_data:") {
			t.Error("Should have prometheus_data volume")
		}
		if !strings.Contains(content, "grafana_data:") {
			t.Error("Should have grafana_data volume")
		}
	})

	t.Run("custom tag", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Tag = "my-test"

		content, err := GenerateComposeFile(cfg)
		if err != nil {
			t.Fatalf("GenerateComposeFile() error = %v", err)
		}

		if !strings.Contains(content, "milvus-etcd-my-test") {
			t.Error("Container names should use custom tag")
		}
		if !strings.Contains(content, "milvus-standalone-my-test") {
			t.Error("Container names should use custom tag")
		}
	})

	t.Run("custom version", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MilvusVersion = "v2.4.0"

		content, err := GenerateComposeFile(cfg)
		if err != nil {
			t.Fatalf("GenerateComposeFile() error = %v", err)
		}

		if !strings.Contains(content, "milvusdb/milvus:v2.4.0") {
			t.Error("Should use custom Milvus version")
		}
	})

	t.Run("custom ports", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MilvusPort = 29530
		cfg.MinioPort = 19000
		cfg.MinioConsole = 19001

		content, err := GenerateComposeFile(cfg)
		if err != nil {
			t.Fatalf("GenerateComposeFile() error = %v", err)
		}

		if !strings.Contains(content, "29530:19530") {
			t.Error("Should use custom Milvus port")
		}
		if !strings.Contains(content, "19000:9000") {
			t.Error("Should use custom MinIO port")
		}
		if !strings.Contains(content, "19001:9001") {
			t.Error("Should use custom MinIO console port")
		}
	})
}

func TestGenerateComposeFile_Structure(t *testing.T) {
	cfg := DefaultConfig()
	content, err := GenerateComposeFile(cfg)
	if err != nil {
		t.Fatalf("GenerateComposeFile() error = %v", err)
	}

	// Check for required sections
	requiredSections := []string{
		"services:",
		"networks:",
		"volumes:",
	}

	for _, section := range requiredSections {
		if !strings.Contains(content, section) {
			t.Errorf("Should contain %s section", section)
		}
	}

	// Check for network configuration
	if !strings.Contains(content, "driver: bridge") {
		t.Error("Should have bridge network driver")
	}

	// Check for core volumes
	coreVolumes := []string{"etcd_data:", "minio_data:", "milvus_data:"}
	for _, vol := range coreVolumes {
		if !strings.Contains(content, vol) {
			t.Errorf("Should contain %s volume", vol)
		}
	}
}

func TestGeneratePrometheusConfig(t *testing.T) {
	cfg := DefaultConfig()
	content := GeneratePrometheusConfig(cfg)

	if content == "" {
		t.Error("GeneratePrometheusConfig should return non-empty string")
	}

	// Check for required sections
	if !strings.Contains(content, "global:") {
		t.Error("Should contain global section")
	}
	if !strings.Contains(content, "scrape_configs:") {
		t.Error("Should contain scrape_configs section")
	}
	if !strings.Contains(content, "job_name: 'milvus'") {
		t.Error("Should contain milvus job")
	}
	if !strings.Contains(content, "standalone:9091") {
		t.Error("Should target standalone on metrics port")
	}
}
