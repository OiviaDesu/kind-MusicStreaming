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
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	musicv1 "github.com/example/managedapp-operator/api/v1"
)

// Manager handles status updates for MusicService objects
type Manager struct {
	client client.Client
}

// NewManager creates a new status manager
func NewManager(c client.Client) *Manager {
	return &Manager{client: c}
}

// setCondition adds or updates a condition in the conditions slice
func setCondition(conditions *[]metav1.Condition, condition metav1.Condition) {
	if conditions == nil || *conditions == nil {
		*conditions = make([]metav1.Condition, 0, 1)
	}

	// Ensure LastTransitionTime is set
	now := metav1.NewTime(time.Now())
	if condition.LastTransitionTime.IsZero() {
		condition.LastTransitionTime = now
	}

	for i, c := range *conditions {
		if c.Type == condition.Type {
			// Only update LastTransitionTime if status changed
			if c.Status != condition.Status {
				condition.LastTransitionTime = now
			} else {
				condition.LastTransitionTime = c.LastTransitionTime
			}
			(*conditions)[i] = condition
			return
		}
	}
	*conditions = append(*conditions, condition)
}

// UpdateReconciled marks the service as successfully reconciled
func (m *Manager) UpdateReconciled(ctx context.Context, ms *musicv1.MusicService) error {
	setCondition(&ms.Status.Conditions, metav1.Condition{
		Type:               "Reconciled",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: ms.Generation,
		Reason:             "ReconcileSuccess",
		Message:            "Successfully reconciled",
	})

	ms.Status.LastReconcileTime = &metav1.Time{Time: time.Now()}
	ms.Status.LastError = ""

	return m.client.Status().Update(ctx, ms)
}

// UpdateError marks the service with an error condition
func (m *Manager) UpdateError(ctx context.Context, ms *musicv1.MusicService, reason, message string) error {
	ms.Status.Phase = "Failed"
	ms.Status.LastError = message
	ms.Status.LastReconcileTime = &metav1.Time{Time: time.Now()}

	setCondition(&ms.Status.Conditions, metav1.Condition{
		Type:               "Reconciled",
		Status:             metav1.ConditionFalse,
		ObservedGeneration: ms.Generation,
		Reason:             reason,
		Message:            message,
	})

	return m.client.Status().Update(ctx, ms)
}

// UpdateFromAppStatefulSet syncs status from the application StatefulSet
func (m *Manager) UpdateFromAppStatefulSet(ctx context.Context, ms *musicv1.MusicService, sts *appsv1.StatefulSet) error {
	ms.Status.ReadyReplicas = sts.Status.ReadyReplicas
	ms.Status.DesiredReplicas = *sts.Spec.Replicas
	ms.Status.ObservedGeneration = ms.Generation

	if sts.Status.ReadyReplicas == 0 {
		ms.Status.Phase = "Pending"
		setCondition(&ms.Status.Conditions, metav1.Condition{
			Type:               "Available",
			Status:             metav1.ConditionFalse,
			ObservedGeneration: ms.Generation,
			Reason:             "PodsNotReady",
			Message:            "Waiting for pods to be ready",
		})
	} else if sts.Status.ReadyReplicas < *sts.Spec.Replicas {
		ms.Status.Phase = "Progressing"
		setCondition(&ms.Status.Conditions, metav1.Condition{
			Type:               "Available",
			Status:             metav1.ConditionFalse,
			ObservedGeneration: ms.Generation,
			Reason:             "PodsProgressing",
			Message:            fmt.Sprintf("Waiting for pods: %d/%d ready", sts.Status.ReadyReplicas, *sts.Spec.Replicas),
		})
	} else {
		ms.Status.Phase = "Available"
		setCondition(&ms.Status.Conditions, metav1.Condition{
			Type:               "Available",
			Status:             metav1.ConditionTrue,
			ObservedGeneration: ms.Generation,
			Reason:             "PodsReady",
			Message:            "All replicas are ready",
		})
	}

	m.updateStorageWarnings(ctx, ms, sts, "music-data", ms.Name, ms.Spec.Storage.Size, "StorageWarningApp")

	return m.client.Status().Update(ctx, ms)
}

// UpdateDatabase updates database-specific status
func (m *Manager) UpdateDatabase(ctx context.Context, ms *musicv1.MusicService) error {
	if ms.Status.Database == nil {
		ms.Status.Database = &musicv1.DatabaseStatus{}
	}

	// Check master status
	masterSts := &appsv1.StatefulSet{}
	masterName := types.NamespacedName{Name: ms.Name + "-db-master", Namespace: ms.Namespace}
	if err := m.client.Get(ctx, masterName, masterSts); err == nil {
		ms.Status.Database.MasterReady = masterSts.Status.ReadyReplicas > 0

		if masterSts.Status.ReadyReplicas > 0 {
			ms.Status.Database.Phase = "Ready"
		} else {
			ms.Status.Database.Phase = "Pending"
		}

		if ms.Spec.Database.Storage != nil {
			m.updateStorageWarnings(ctx, ms, masterSts, "db-data", ms.Name+"-db-master", ms.Spec.Database.Storage.Size, "StorageWarningDatabase")
		}
	}

	// Check replica status
	if ms.Spec.Database.Replicas > 0 {
		replicaSts := &appsv1.StatefulSet{}
		replicaName := types.NamespacedName{Name: ms.Name + "-db-replica", Namespace: ms.Namespace}
		if err := m.client.Get(ctx, replicaName, replicaSts); err == nil {
			ms.Status.Database.ReplicasReady = replicaSts.Status.ReadyReplicas
			ms.Status.Database.ReplicaEverCreated = true
			ms.Status.Database.ReplicaDeletionDetected = false
			ms.Status.Database.ReplicaLastSeen = &metav1.Time{Time: time.Now()}
			ms.Status.Database.ReplicationReady = replicaSts.Status.ReadyReplicas > 0

			setCondition(&ms.Status.Conditions, metav1.Condition{
				Type:               "DatabaseReplicaHistory",
				Status:             metav1.ConditionTrue,
				ObservedGeneration: ms.Generation,
				Reason:             "ReplicaObserved",
				Message:            "Replica StatefulSet is present",
			})
		} else if errors.IsNotFound(err) {
			if ms.Status.Database.ReplicaEverCreated {
				ms.Status.Database.ReplicaDeletionDetected = true
				setCondition(&ms.Status.Conditions, metav1.Condition{
					Type:               "DatabaseReplicaHistory",
					Status:             metav1.ConditionFalse,
					ObservedGeneration: ms.Generation,
					Reason:             "ReplicaDeleted",
					Message:            "Replica StatefulSet was deleted after previously existing",
				})
			}
		}
	}

	return m.client.Status().Update(ctx, ms)
}

func (m *Manager) updateStorageWarnings(ctx context.Context, ms *musicv1.MusicService, sts *appsv1.StatefulSet, claimName, appName, desiredSize, conditionType string) {
	currentSize, hasCurrent := storageRequestFromStatefulSet(sts)
	if hasCurrent && desiredSize != "" {
		desired, err := resource.ParseQuantity(desiredSize)
		if err == nil {
			if desired.Cmp(currentSize) < 0 {
				setCondition(&ms.Status.Conditions, metav1.Condition{
					Type:               conditionType,
					Status:             metav1.ConditionFalse,
					ObservedGeneration: ms.Generation,
					Reason:             "ShrinkNotSupported",
					Message:            "Requested storage size is smaller than current PVC size",
				})
				return
			}
		}
	}

	if pvcs, err := m.listPVCsByPrefix(ctx, claimName, appName, ms.Namespace); err == nil {
		for _, pvc := range pvcs {
			if pvc.Status.Phase != corev1.ClaimBound {
				setCondition(&ms.Status.Conditions, metav1.Condition{
					Type:               conditionType,
					Status:             metav1.ConditionFalse,
					ObservedGeneration: ms.Generation,
					Reason:             "PVCNotBound",
					Message:            "One or more PVCs are not bound yet",
				})
				return
			}
		}
	}

	setCondition(&ms.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: ms.Generation,
		Reason:             "StorageHealthy",
		Message:            "Storage requests are within expected bounds",
	})
}

func storageRequestFromStatefulSet(sts *appsv1.StatefulSet) (resource.Quantity, bool) {
	if len(sts.Spec.VolumeClaimTemplates) == 0 {
		return resource.Quantity{}, false
	}
	requests := sts.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests
	storage, ok := requests[corev1.ResourceStorage]
	return storage, ok
}

func (m *Manager) listPVCsByPrefix(ctx context.Context, claimName, appName, namespace string) ([]corev1.PersistentVolumeClaim, error) {
	pvcList := &corev1.PersistentVolumeClaimList{}
	if err := m.client.List(ctx, pvcList, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, err
	}

	prefix := fmt.Sprintf("%s-%s-", claimName, appName)
	filtered := make([]corev1.PersistentVolumeClaim, 0, len(pvcList.Items))
	for _, pvc := range pvcList.Items {
		if len(pvc.Name) >= len(prefix) && pvc.Name[:len(prefix)] == prefix {
			filtered = append(filtered, pvc)
		}
	}

	return filtered, nil
}
