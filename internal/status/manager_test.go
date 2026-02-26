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

package status

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	musicv1 "github.com/example/managedapp-operator/api/v1"
)

// newValidMusicService creates a MusicService with valid required fields
func newValidMusicService(name string) *musicv1.MusicService {
	return &musicv1.MusicService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: musicv1.MusicServiceSpec{
			Replicas: 1,
			Image:    "test:latest",
			Port:     8080,
			Storage: musicv1.StorageSpec{
				Size: "10Gi",
			},
			Streaming: musicv1.StreamingSpec{
				Bitrate:        "128k",
				MaxConnections: 100,
			},
		},
	}
}

func TestSetCondition(t *testing.T) {
	tests := []struct {
		name       string
		conditions *[]metav1.Condition
		condition  metav1.Condition
		wantLen    int
	}{
		{
			name:       "add condition to empty slice",
			conditions: &[]metav1.Condition{},
			condition: metav1.Condition{
				Type:   "Available",
				Status: metav1.ConditionTrue,
				Reason: "TestReason",
			},
			wantLen: 1,
		},
		{
			name: "update existing condition",
			conditions: &[]metav1.Condition{
				{
					Type:   "Available",
					Status: metav1.ConditionFalse,
					Reason: "OldReason",
				},
			},
			condition: metav1.Condition{
				Type:   "Available",
				Status: metav1.ConditionTrue,
				Reason: "NewReason",
			},
			wantLen: 1,
		},
		{
			name: "add different condition type",
			conditions: &[]metav1.Condition{
				{
					Type:   "Available",
					Status: metav1.ConditionTrue,
					Reason: "TestReason",
				},
			},
			condition: metav1.Condition{
				Type:   "Reconciled",
				Status: metav1.ConditionTrue,
				Reason: "TestReason",
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setCondition(tt.conditions, tt.condition)

			if len(*tt.conditions) != tt.wantLen {
				t.Errorf("got %d conditions, want %d", len(*tt.conditions), tt.wantLen)
			}

			found := false
			for _, c := range *tt.conditions {
				if c.Type == tt.condition.Type {
					found = true
					if c.Status != tt.condition.Status {
						t.Errorf("got status %v, want %v", c.Status, tt.condition.Status)
					}
					if !c.LastTransitionTime.IsZero() {
						break
					}
					t.Error("LastTransitionTime not set")
				}
			}

			if !found {
				t.Error("condition not found")
			}
		})
	}
}

func TestStatusManager(t *testing.T) {
	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{"../../config/crd/bases"},
	}

	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatalf("failed to start test environment: %v", err)
	}

	defer func() {
		_ = testEnv.Stop()
	}()

	err = musicv1.AddToScheme(scheme.Scheme)
	if err != nil {
		t.Fatalf("failed to add scheme: %v", err)
	}

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()
	manager := NewManager(k8sClient)

	t.Run("UpdateReconciled should set Reconciled condition", func(t *testing.T) {
		ms := newValidMusicService("test-reconciled")

		if err := k8sClient.Create(ctx, ms); err != nil {
			t.Fatalf("failed to create MusicService: %v", err)
		}

		err := manager.UpdateReconciled(ctx, ms)
		if err != nil {
			t.Fatalf("UpdateReconciled failed: %v", err)
		}

		updated := &musicv1.MusicService{}
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ms), updated); err != nil {
			t.Fatalf("failed to get updated MusicService: %v", err)
		}

		found := false
		for _, cond := range updated.Status.Conditions {
			if cond.Type == "Reconciled" {
				found = true
				if cond.Status != metav1.ConditionTrue {
					t.Errorf("expected Reconciled condition to be True, got %v", cond.Status)
				}
				break
			}
		}

		if !found {
			t.Error("Reconciled condition not found")
		}
	})

	t.Run("UpdateError should set Reconciled condition to False", func(t *testing.T) {
		ms := newValidMusicService("test-error")

		if err := k8sClient.Create(ctx, ms); err != nil {
			t.Fatalf("failed to create MusicService: %v", err)
		}

		err := manager.UpdateError(ctx, ms, "TestError", "Test error message")
		if err != nil {
			t.Fatalf("UpdateError failed: %v", err)
		}

		updated := &musicv1.MusicService{}
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ms), updated); err != nil {
			t.Fatalf("failed to get updated MusicService: %v", err)
		}

		if updated.Status.Phase != "Failed" {
			t.Errorf("expected phase Failed, got %s", updated.Status.Phase)
		}

		found := false
		for _, cond := range updated.Status.Conditions {
			if cond.Type == "Reconciled" && cond.Status == metav1.ConditionFalse {
				found = true
				break
			}
		}

		if !found {
			t.Error("Failed Reconciled condition not found")
		}
	})

	t.Run("UpdateFromAppStatefulSet should update replica status", func(t *testing.T) {
		ms := newValidMusicService("test-sts-status")
		ms.Spec.Replicas = 3

		if err := k8sClient.Create(ctx, ms); err != nil {
			t.Fatalf("failed to create MusicService: %v", err)
		}

		// Create a mock StatefulSet with status (not persisting to cluster for this test)
		sts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-sts-status",
				Namespace: "default",
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: int32Ptr(3),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "test"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test"}},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "test", Image: "test:latest"}}},
				},
			},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas: 2,
			},
		}

		// Update manager with the mock StatefulSet status
		err := manager.UpdateFromAppStatefulSet(ctx, ms, sts)
		if err != nil {
			t.Fatalf("UpdateFromAppStatefulSet failed: %v", err)
		}

		updated := &musicv1.MusicService{}
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ms), updated); err != nil {
			t.Fatalf("failed to get updated MusicService: %v", err)
		}

		if updated.Status.ReadyReplicas != 2 {
			t.Errorf("expected 2 ready replicas, got %d", updated.Status.ReadyReplicas)
		}

		if updated.Status.Phase != "Progressing" {
			t.Errorf("expected phase Progressing, got %s", updated.Status.Phase)
		}
	})
}

func int32Ptr(i int32) *int32 {
	return &i
}
