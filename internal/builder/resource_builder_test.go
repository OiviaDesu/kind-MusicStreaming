/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package builder

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	musicv1 "github.com/example/managedapp-operator/api/v1"
)

func TestResourceBuilder(t *testing.T) {
	builder := NewResourceBuilder(scheme.Scheme)

	tests := []struct {
		name   string
		ms     *musicv1.MusicService
		testFn func(*testing.T, *musicv1.MusicService, *ResourceBuilder)
	}{
		{
			name: "BuildAppStatefulSet creates valid StatefulSet",
			ms: &musicv1.MusicService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "default",
				},
				Spec: musicv1.MusicServiceSpec{
					Replicas: 2,
					Image:    "nginx:latest",
					Port:     8080,
					Storage: musicv1.StorageSpec{
						Size:         "10Gi",
						UpdatePolicy: "Recreate",
					},
					Streaming: musicv1.StreamingSpec{
						Bitrate:        "320k",
						MaxConnections: 1000,
					},
					Resources: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
					},
				},
			},
			testFn: func(t *testing.T, ms *musicv1.MusicService, rb *ResourceBuilder) {
				sts := rb.BuildAppStatefulSet(ms)

				if sts == nil {
					t.Fatal("BuildAppStatefulSet returned nil")
				}

				if sts.Name != "test-app" {
					t.Errorf("expected name test-app, got %s", sts.Name)
				}

				if sts.Namespace != "default" {
					t.Errorf("expected namespace default, got %s", sts.Namespace)
				}

				if *sts.Spec.Replicas != 2 {
					t.Errorf("expected 2 replicas, got %d", *sts.Spec.Replicas)
				}

				if len(sts.Spec.Template.Spec.Containers) != 1 {
					t.Errorf("expected 1 container, got %d", len(sts.Spec.Template.Spec.Containers))
				}

				container := sts.Spec.Template.Spec.Containers[0]
				if container.Image != "nginx:latest" {
					t.Errorf("expected image nginx:latest, got %s", container.Image)
				}

				if len(sts.Spec.VolumeClaimTemplates) != 1 {
					t.Errorf("expected 1 volume claim template, got %d", len(sts.Spec.VolumeClaimTemplates))
				}
			},
		},

		{
			name: "BuildAppService creates valid Service",
			ms: &musicv1.MusicService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-svc",
					Namespace: "default",
				},
				Spec: musicv1.MusicServiceSpec{
					Replicas: 1,
					Image:    "test:latest",
					Port:     9000,
				},
			},
			testFn: func(t *testing.T, ms *musicv1.MusicService, rb *ResourceBuilder) {
				svc := rb.BuildAppService(ms)

				if svc == nil {
					t.Fatal("BuildAppService returned nil")
				}

				if svc.Name != "test-svc" {
					t.Errorf("expected name test-svc, got %s", svc.Name)
				}

				if svc.Spec.Type != corev1.ServiceTypeClusterIP {
					t.Errorf("expected ClusterIP service type, got %s", svc.Spec.Type)
				}

				if len(svc.Spec.Ports) != 1 {
					t.Errorf("expected 1 port, got %d", len(svc.Spec.Ports))
				}

				if svc.Spec.Ports[0].Port != 9000 {
					t.Errorf("expected port 9000, got %d", svc.Spec.Ports[0].Port)
				}
			},
		},

		{
			name: "BuildDatabaseMasterStatefulSet creates master database",
			ms: &musicv1.MusicService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-db",
					Namespace: "default",
				},
				Spec: musicv1.MusicServiceSpec{
					Database: &musicv1.DatabaseSpec{
						Enabled:      true,
						Image:        "mariadb:10.11",
						RootPassword: "secret",
						Storage: &musicv1.StorageSpec{
							Size:         "20Gi",
							UpdatePolicy: "Recreate",
						},
					},
				},
			},
			testFn: func(t *testing.T, ms *musicv1.MusicService, rb *ResourceBuilder) {
				sts := rb.BuildDatabaseMasterStatefulSet(ms)

				if sts == nil {
					t.Fatal("BuildDatabaseMasterStatefulSet returned nil")
				}

				if sts.Name != "test-db-db-master" {
					t.Errorf("expected name test-db-db-master, got %s", sts.Name)
				}

				if *sts.Spec.Replicas != 1 {
					t.Errorf("expected 1 replica for master, got %d", *sts.Spec.Replicas)
				}

				if len(sts.Spec.Template.Spec.Containers) == 0 {
					t.Fatal("no containers in master StatefulSet")
				}

				container := sts.Spec.Template.Spec.Containers[0]
				if container.Image != "mariadb:10.11" {
					t.Errorf("expected image mariadb:10.11, got %s", container.Image)
				}
			},
		},

		{
			name: "BuildDatabaseReplicaStatefulSet creates replica database",
			ms: &musicv1.MusicService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-db-replica",
					Namespace: "default",
				},
				Spec: musicv1.MusicServiceSpec{
					Database: &musicv1.DatabaseSpec{
						Enabled:      true,
						Replicas:     2,
						Image:        "mariadb:10.11",
						RootPassword: "secret",
						Storage: &musicv1.StorageSpec{
							Size:         "20Gi",
							UpdatePolicy: "Recreate",
						},
						Replication: &musicv1.DatabaseReplicationSpec{
							Enabled: boolPtr(true),
							GTID:    boolPtr(true),
						},
					},
				},
			},
			testFn: func(t *testing.T, ms *musicv1.MusicService, rb *ResourceBuilder) {
				sts := rb.BuildDatabaseReplicaStatefulSet(ms)

				if sts == nil {
					t.Fatal("BuildDatabaseReplicaStatefulSet returned nil")
				}

				if sts.Name != "test-db-replica-db-replica" {
					t.Errorf("expected name test-db-replica-db-replica, got %s", sts.Name)
				}

				if *sts.Spec.Replicas != 2 {
					t.Errorf("expected 2 replicas, got %d", *sts.Spec.Replicas)
				}
			},
		},
		{
			name: "BuildDatabaseGaleraStatefulSet creates Galera Cluster StatefulSet",
			ms: &musicv1.MusicService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ha",
					Namespace: "default",
				},
				Spec: musicv1.MusicServiceSpec{
					Database: &musicv1.DatabaseSpec{
						Enabled:      true,
						Replicas:     2,
						Image:        "mariadb:10.11",
						RootPassword: "secret",
						Storage: &musicv1.StorageSpec{
							Size:         "20Gi",
							UpdatePolicy: "Recreate",
						},
						HighAvailability: &musicv1.DatabaseHighAvailabilitySpec{
							Enabled: true,
						},
					},
				},
			},
			testFn: func(t *testing.T, ms *musicv1.MusicService, rb *ResourceBuilder) {
				sts := rb.BuildDatabaseGaleraStatefulSet(ms)

				if sts == nil {
					t.Fatal("BuildDatabaseGaleraStatefulSet returned nil")
				}

				if sts.Name != "test-ha-db-galera" {
					t.Errorf("expected name test-ha-db-galera, got %s", sts.Name)
				}

				// totalReplicas = replicas (2) + 1 = 3
				if *sts.Spec.Replicas != 3 {
					t.Errorf("expected 3 replicas (1 primary + 2), got %d", *sts.Spec.Replicas)
				}

				// Headless service name must match StatefulSet name
				if sts.Spec.ServiceName != "test-ha-db-galera" {
					t.Errorf("expected headless service name test-ha-db-galera, got %s", sts.Spec.ServiceName)
				}

				// Verify Galera ports are exposed
				if len(sts.Spec.Template.Spec.Containers) == 0 {
					t.Fatal("no containers in Galera StatefulSet")
				}
				container := sts.Spec.Template.Spec.Containers[0]
				portNames := make(map[string]bool)
				for _, p := range container.Ports {
					portNames[p.Name] = true
				}
				for _, expected := range []string{"mysql", "galera-repl", "galera-ist", "galera-sst"} {
					if !portNames[expected] {
						t.Errorf("expected Galera port %s to be present", expected)
					}
				}

				// Verify init container exists for Galera config
				if len(sts.Spec.Template.Spec.InitContainers) == 0 {
					t.Fatal("no init containers in Galera StatefulSet")
				}
				if sts.Spec.Template.Spec.InitContainers[0].Name != "init-galera-config" {
					t.Errorf("expected init container name init-galera-config, got %s", sts.Spec.Template.Spec.InitContainers[0].Name)
				}

				// Verify component label is db-galera
				if sts.Spec.Template.Labels["component"] != "db-galera" {
					t.Errorf("expected component label db-galera, got %s", sts.Spec.Template.Labels["component"])
				}
			},
		},
		{
			name: "BuildDatabaseGaleraService creates headless service",
			ms: &musicv1.MusicService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ha",
					Namespace: "default",
				},
				Spec: musicv1.MusicServiceSpec{
					Database: &musicv1.DatabaseSpec{
						Enabled: true,
					},
				},
			},
			testFn: func(t *testing.T, ms *musicv1.MusicService, rb *ResourceBuilder) {
				svc := rb.BuildDatabaseGaleraService(ms)

				if svc == nil {
					t.Fatal("BuildDatabaseGaleraService returned nil")
				}

				if svc.Name != "test-ha-db-galera" {
					t.Errorf("expected name test-ha-db-galera, got %s", svc.Name)
				}

				if svc.Spec.ClusterIP != "None" {
					t.Errorf("expected headless service (ClusterIP=None), got %s", svc.Spec.ClusterIP)
				}

				if !svc.Spec.PublishNotReadyAddresses {
					t.Error("expected PublishNotReadyAddresses=true for Galera cluster discovery")
				}
			},
		},
		{
			name: "BuildDatabaseGaleraPrimaryService creates HA write service",
			ms: &musicv1.MusicService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ha",
					Namespace: "default",
				},
				Spec: musicv1.MusicServiceSpec{
					Database: &musicv1.DatabaseSpec{
						Enabled: true,
					},
				},
			},
			testFn: func(t *testing.T, ms *musicv1.MusicService, rb *ResourceBuilder) {
				svc := rb.BuildDatabaseGaleraPrimaryService(ms)

				if svc == nil {
					t.Fatal("BuildDatabaseGaleraPrimaryService returned nil")
				}

				// Uses same name as master service for backward compatibility
				if svc.Name != "test-ha-db-master" {
					t.Errorf("expected name test-ha-db-master, got %s", svc.Name)
				}

				// Must select all galera nodes (not just pod-0) for HA
				if svc.Spec.Selector["component"] != "db-galera" {
					t.Errorf("expected selector component=db-galera for HA, got %s", svc.Spec.Selector["component"])
				}

				// Must NOT be headless (needs load-balancing across healthy nodes)
				if svc.Spec.ClusterIP == "None" {
					t.Error("primary service should not be headless; must route to healthy nodes")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFn(t, tt.ms, builder)
		})
	}
}

// Helper functions

func boolPtr(b bool) *bool {
	return &b
}
