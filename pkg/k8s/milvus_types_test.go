package k8s

import (
	"encoding/json"
	"testing"
)

func TestConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"MilvusGroup", MilvusGroup, "milvus.io"},
		{"MilvusVersion", MilvusVersion, "v1beta1"},
		{"MilvusResource", MilvusResource, "milvuses"},
		{"MilvusKind", MilvusKind, "Milvus"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %s, want %s", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestMilvusMode(t *testing.T) {
	if MilvusModeStandalone != "standalone" {
		t.Errorf("MilvusModeStandalone = %s, want standalone", MilvusModeStandalone)
	}
	if MilvusModeCluster != "cluster" {
		t.Errorf("MilvusModeCluster = %s, want cluster", MilvusModeCluster)
	}
}

func TestMilvusSpec_JSON(t *testing.T) {
	spec := MilvusSpec{
		Mode: MilvusModeStandalone,
		Components: MilvusComponents{
			Image: "milvusdb/milvus:v2.4.0",
		},
		Config: map[string]any{
			"log.level": "debug",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Unmarshal back
	var decoded MilvusSpec
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Mode != spec.Mode {
		t.Errorf("Mode = %s, want %s", decoded.Mode, spec.Mode)
	}
	if decoded.Components.Image != spec.Components.Image {
		t.Errorf("Components.Image = %s, want %s", decoded.Components.Image, spec.Components.Image)
	}
}

func TestMilvusDependencies_JSON(t *testing.T) {
	deps := MilvusDependencies{
		Etcd: EtcdConfig{
			External:  true,
			Endpoints: []string{"etcd-0:2379", "etcd-1:2379"},
		},
		Storage: StorageConfig{
			Type:     "MinIO",
			External: false,
		},
		MsgStreamType: "pulsar",
	}

	data, err := json.Marshal(deps)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded MilvusDependencies
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Etcd.External != true {
		t.Error("Etcd.External should be true")
	}
	if len(decoded.Etcd.Endpoints) != 2 {
		t.Errorf("Etcd.Endpoints length = %d, want 2", len(decoded.Etcd.Endpoints))
	}
	if decoded.Storage.Type != "MinIO" {
		t.Errorf("Storage.Type = %s, want MinIO", decoded.Storage.Type)
	}
	if decoded.MsgStreamType != "pulsar" {
		t.Errorf("MsgStreamType = %s, want pulsar", decoded.MsgStreamType)
	}
}

func TestComponentSpec_JSON(t *testing.T) {
	replicas := int32(3)
	spec := ComponentSpec{
		Replicas: &replicas,
		Resources: &ResourceRequirements{
			Limits: map[string]string{
				"cpu":    "4",
				"memory": "8Gi",
			},
			Requests: map[string]string{
				"cpu":    "2",
				"memory": "4Gi",
			},
		},
		NodeSelector: map[string]string{
			"disktype": "ssd",
		},
	}

	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded ComponentSpec
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Replicas == nil || *decoded.Replicas != 3 {
		t.Error("Replicas should be 3")
	}
	if decoded.Resources == nil {
		t.Fatal("Resources should not be nil")
	}
	if decoded.Resources.Limits["cpu"] != "4" {
		t.Errorf("Resources.Limits[cpu] = %s, want 4", decoded.Resources.Limits["cpu"])
	}
	if decoded.NodeSelector["disktype"] != "ssd" {
		t.Errorf("NodeSelector[disktype] = %s, want ssd", decoded.NodeSelector["disktype"])
	}
}

func TestMilvusStatus_JSON(t *testing.T) {
	status := MilvusStatus{
		Status:   "Healthy",
		Endpoint: "milvus.default.svc.cluster.local:19530",
		ComponentsDeployStatus: map[string]ComponentDeployStatus{
			"queryNode": {
				Generation: 1,
				Image:      "milvusdb/milvus:v2.4.0",
				Status: DeploymentStatus{
					Replicas:          3,
					ReadyReplicas:     3,
					AvailableReplicas: 3,
					UpdatedReplicas:   3,
				},
			},
		},
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded MilvusStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Status != "Healthy" {
		t.Errorf("Status = %s, want Healthy", decoded.Status)
	}
	if decoded.Endpoint != status.Endpoint {
		t.Errorf("Endpoint = %s, want %s", decoded.Endpoint, status.Endpoint)
	}

	qn, ok := decoded.ComponentsDeployStatus["queryNode"]
	if !ok {
		t.Fatal("queryNode not found in ComponentsDeployStatus")
	}
	if qn.Status.ReadyReplicas != 3 {
		t.Errorf("queryNode.Status.ReadyReplicas = %d, want 3", qn.Status.ReadyReplicas)
	}
}

func TestInClusterConfig_JSON(t *testing.T) {
	cfg := InClusterConfig{
		DeletionPolicy: "Retain",
		PVCDeletion:    false,
		Values: map[string]any{
			"replicaCount": 3,
		},
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded InClusterConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.DeletionPolicy != "Retain" {
		t.Errorf("DeletionPolicy = %s, want Retain", decoded.DeletionPolicy)
	}
	if decoded.PVCDeletion != false {
		t.Error("PVCDeletion should be false")
	}
}

func TestVolume_JSON(t *testing.T) {
	vol := Volume{
		Name: "tls-certs",
		Secret: &SecretSource{
			SecretName: "milvus-tls-secret",
		},
	}

	data, err := json.Marshal(vol)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded Volume
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Name != "tls-certs" {
		t.Errorf("Name = %s, want tls-certs", decoded.Name)
	}
	if decoded.Secret == nil {
		t.Fatal("Secret should not be nil")
	}
	if decoded.Secret.SecretName != "milvus-tls-secret" {
		t.Errorf("Secret.SecretName = %s, want milvus-tls-secret", decoded.Secret.SecretName)
	}
}

func TestVolumeMount_JSON(t *testing.T) {
	mount := VolumeMount{
		Name:      "tls-certs",
		MountPath: "/etc/milvus/tls",
		ReadOnly:  true,
	}

	data, err := json.Marshal(mount)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded VolumeMount
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Name != "tls-certs" {
		t.Errorf("Name = %s, want tls-certs", decoded.Name)
	}
	if decoded.MountPath != "/etc/milvus/tls" {
		t.Errorf("MountPath = %s, want /etc/milvus/tls", decoded.MountPath)
	}
	if decoded.ReadOnly != true {
		t.Error("ReadOnly should be true")
	}
}

func TestMilvusList_JSON(t *testing.T) {
	list := MilvusList{
		Items: []Milvus{
			{
				Spec: MilvusSpec{
					Mode: MilvusModeStandalone,
				},
			},
			{
				Spec: MilvusSpec{
					Mode: MilvusModeCluster,
				},
			},
		},
	}

	data, err := json.Marshal(list)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded MilvusList
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(decoded.Items) != 2 {
		t.Errorf("Items length = %d, want 2", len(decoded.Items))
	}
	if decoded.Items[0].Spec.Mode != MilvusModeStandalone {
		t.Errorf("Items[0].Spec.Mode = %s, want standalone", decoded.Items[0].Spec.Mode)
	}
	if decoded.Items[1].Spec.Mode != MilvusModeCluster {
		t.Errorf("Items[1].Spec.Mode = %s, want cluster", decoded.Items[1].Spec.Mode)
	}
}
