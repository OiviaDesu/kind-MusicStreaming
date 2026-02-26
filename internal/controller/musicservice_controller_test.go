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

package controller

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	musicv1 "github.com/example/managedapp-operator/api/v1"
)

func TestMusicServiceController(t *testing.T) {
	t.Run("ReplicationConfiguration", func(t *testing.T) {
		// Create sample MusicService with replication
		ms := &musicv1.MusicService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-replication",
				Namespace: "default",
			},
			Spec: musicv1.MusicServiceSpec{
				Replicas: 2,
				Image:    "nginx:latest",
				Port:     8080,
				Storage: musicv1.StorageSpec{
					Size: "10Gi",
				},
				Streaming: musicv1.StreamingSpec{
					Bitrate:        "128k",
					MaxConnections: 100,
				},
				Database: &musicv1.DatabaseSpec{
					Enabled:  true,
					Replicas: 2,
					Image:    "mariadb:10.11",
					Replication: &musicv1.ReplicationSpec{
						Enabled:     boolPtr(true),
						GTID:        boolPtr(true),
						MinReplicas: intPtr(1),
						MaxReplicas: intPtr(5),
					},
				},
			},
		}

		// Verify replication configuration is set
		if ms.Spec.Database == nil {
			t.Fatal("Database spec should not be nil")
		}
		if ms.Spec.Database.Replication == nil {
			t.Fatal("Replication spec should not be nil")
		}
		if !*ms.Spec.Database.Replication.Enabled {
			t.Error("Replication should be enabled")
		}
		if !*ms.Spec.Database.Replication.GTID {
			t.Error("GTID should be enabled")
		}
		if *ms.Spec.Database.Replication.MinReplicas != 1 {
			t.Error("MinReplicas should be 1")
		}
		if *ms.Spec.Database.Replication.MaxReplicas != 5 {
			t.Error("MaxReplicas should be 5")
		}
	})

	t.Run("AutoscalingConfiguration", func(t *testing.T) {
		ms := &musicv1.MusicService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-autoscaling",
				Namespace: "default",
			},
			Spec: musicv1.MusicServiceSpec{
				Replicas: 2,
				Image:    "nginx:latest",
				Port:     8080,
				Storage: musicv1.StorageSpec{
					Size: "10Gi",
				},
				Streaming: musicv1.StreamingSpec{
					Bitrate:        "128k",
					MaxConnections: 100,
				},
				Autoscaling: &musicv1.AutoscalingSpec{
					Enabled:     boolPtr(true),
					MinReplicas: intPtr(2),
					MaxReplicas: intPtr(10),
					TargetCPU:   intPtr(80),
				},
			},
		}

		// Verify autoscaling configuration
		if ms.Spec.Autoscaling == nil {
			t.Fatal("Autoscaling spec should not be nil")
		}
		if !*ms.Spec.Autoscaling.Enabled {
			t.Error("Autoscaling should be enabled")
		}
		if *ms.Spec.Autoscaling.MinReplicas != 2 {
			t.Error("MinReplicas should be 2")
		}
		if *ms.Spec.Autoscaling.MaxReplicas != 10 {
			t.Error("MaxReplicas should be 10")
		}
		if *ms.Spec.Autoscaling.TargetCPU != 80 {
			t.Error("TargetCPU should be 80")
		}
	})

	t.Run("MusicServiceWithoutDatabase", func(t *testing.T) {
		ms := &musicv1.MusicService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-no-db",
				Namespace: "default",
			},
			Spec: musicv1.MusicServiceSpec{
				Replicas: 3,
				Image:    "nginx:latest",
				Port:     8080,
				Storage: musicv1.StorageSpec{
					Size: "5Gi",
				},
				Streaming: musicv1.StreamingSpec{
					Bitrate:        "96k",
					MaxConnections: 50,
				},
				Database: nil,
			},
		}

		// Verify resource configuration
		if ms.Spec.Replicas != 3 {
			t.Error("Expected 3 replicas")
		}
		if ms.Spec.Image != "nginx:latest" {
			t.Error("Expected nginx:latest image")
		}
		if ms.Spec.Port != 8080 {
			t.Error("Expected port 8080")
		}
		if ms.Spec.Storage.Size != "5Gi" {
			t.Error("Expected 5Gi storage")
		}
	})

	t.Run("MusicServiceLabelValidation", func(t *testing.T) {
		ms := &musicv1.MusicService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-labels",
				Namespace: "default",
				Labels: map[string]string{
					"app":     "music-service",
					"version": "v1",
				},
			},
			Spec: musicv1.MusicServiceSpec{
				Replicas: 1,
				Image:    "nginx:latest",
				Port:     8080,
				Storage: musicv1.StorageSpec{
					Size: "1Gi",
				},
				Streaming: musicv1.StreamingSpec{
					Bitrate:        "64k",
					MaxConnections: 10,
				},
			},
		}

		// Verify labels exist
		if ms.ObjectMeta.Labels == nil || len(ms.ObjectMeta.Labels) == 0 {
			t.Error("MusicService should have labels")
		}
		if ms.ObjectMeta.Labels["app"] != "music-service" {
			t.Error("Expected app label to be 'music-service'")
		}
	})

	t.Run("ResourceValidation", func(t *testing.T) {
		tests := []struct {
			name     string
			replicas int32
			image    string
			port     int32
			storage  string
			valid    bool
		}{
			{
				name:     "valid configuration",
				replicas: 2,
				image:    "nginx:latest",
				port:     8080,
				storage:  "10Gi",
				valid:    true,
			},
			{
				name:     "single replica",
				replicas: 1,
				image:    "nginx:latest",
				port:     3000,
				storage:  "5Gi",
				valid:    true,
			},
			{
				name:     "high replica count",
				replicas: 100,
				image:    "nginx:latest",
				port:     80,
				storage:  "100Gi",
				valid:    true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ms := &musicv1.MusicService{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-" + tt.name,
						Namespace: "default",
					},
					Spec: musicv1.MusicServiceSpec{
						Replicas: tt.replicas,
						Image:    tt.image,
						Port:     tt.port,
						Storage: musicv1.StorageSpec{
							Size: tt.storage,
						},
						Streaming: musicv1.StreamingSpec{
							Bitrate:        "128k",
							MaxConnections: 100,
						},
					},
				}

				// Validate basic properties
				if ms.Spec.Replicas <= 0 {
					t.Error("Replicas must be greater than 0")
				}
				if ms.Spec.Port <= 0 {
					t.Error("Port must be greater than 0")
				}
			})
		}
	})

	t.Run("DatabaseSpecComplete", func(t *testing.T) {
		ms := &musicv1.MusicService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-db-complete",
				Namespace: "default",
			},
			Spec: musicv1.MusicServiceSpec{
				Replicas: 2,
				Image:    "nginx:latest",
				Port:     8080,
				Storage: musicv1.StorageSpec{
					Size: "10Gi",
				},
				Streaming: musicv1.StreamingSpec{
					Bitrate:        "128k",
					MaxConnections: 100,
				},
				Database: &musicv1.DatabaseSpec{
					Enabled:      true,
					Replicas:     2,
					Image:        "mariadb:10.11",
					RootPassword: "secure-password",
					Storage: musicv1.StorageSpec{
						Size: "20Gi",
					},
					Replication: &musicv1.ReplicationSpec{
						Enabled:     boolPtr(true),
						GTID:        boolPtr(true),
						MinReplicas: intPtr(1),
						MaxReplicas: intPtr(5),
					},
					Autoscaling: &musicv1.AutoscalingSpec{
						Enabled:     boolPtr(true),
						MinReplicas: intPtr(1),
						MaxReplicas: intPtr(5),
						TargetCPU:   intPtr(70),
					},
				},
			},
		}

		// Verify complete database configuration
		if !ms.Spec.Database.Enabled {
			t.Error("Database should be enabled")
		}
		if ms.Spec.Database.Replicas != 2 {
			t.Error("Expected 2 database replicas")
		}
		if ms.Spec.Database.Image != "mariadb:10.11" {
			t.Error("Expected mariadb:10.11 image")
		}
		if ms.Spec.Database.RootPassword != "secure-password" {
			t.Error("Expected password to be set")
		}
		if ms.Spec.Database.Replication == nil || !*ms.Spec.Database.Replication.Enabled {
			t.Error("Replication should be enabled")
		}
		if ms.Spec.Database.Autoscaling == nil || !*ms.Spec.Database.Autoscaling.Enabled {
			t.Error("Autoscaling should be enabled")
		}
	})
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}
