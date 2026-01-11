package embed

import (
	"embed"
)

//go:embed templates/crd/*.yaml
var CRDTemplates embed.FS

// GetCRDTemplate returns the content of a CRD template by name
func GetCRDTemplate(name string) ([]byte, error) {
	return CRDTemplates.ReadFile("templates/crd/" + name + ".yaml")
}

// ListCRDTemplates returns all available CRD template names
func ListCRDTemplates() []string {
	return []string{
		"standalone",
		"standalone-tls",
		"standalone-external-s3",
		"distributed",
		"distributed-ha",
		"distributed-pulsar",
		"distributed-gpu",
	}
}

// CRDTemplateDescriptions returns descriptions for each template
var CRDTemplateDescriptions = map[string]string{
	"standalone":            "Minimal standalone mode for development",
	"standalone-tls":        "Standalone with TLS encryption",
	"standalone-external-s3": "Standalone with external S3/MinIO storage",
	"distributed":           "Production-ready distributed deployment",
	"distributed-ha":        "High availability with coordinator failover",
	"distributed-pulsar":    "Distributed with Pulsar message queue",
	"distributed-gpu":       "Distributed with GPU acceleration",
}
