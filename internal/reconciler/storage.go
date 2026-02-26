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

package reconciler

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	musicv1 "github.com/example/managedapp-operator/api/v1"
)

// Hướng dẫn đọc nhanh:
// - Nếu chưa rõ vì sao cần xử lý storage, xem internal/reconciler/app.go hoặc database.go.
// - Nếu chưa rõ updatePolicy/size, xem api/v1/musicservice_types.go.
// - Nếu chưa rõ luồng tổng thể, xem internal/controller/musicservice_controller.go.

func storageUpdatePolicy(storage musicv1.StorageSpec) musicv1.StorageUpdatePolicy {
	if storage.UpdatePolicy == "" {
		return musicv1.StorageUpdatePolicyResize
	}
	return storage.UpdatePolicy
}

func storageSizeChanged(current, desired *appsv1.StatefulSet) bool {
	currentSize, hasCurrent := storageRequestFromStatefulSet(current)
	desiredSize, hasDesired := storageRequestFromStatefulSet(desired)
	if !hasCurrent || !hasDesired {
		return false
	}
	return currentSize.Cmp(desiredSize) != 0
}

func storageRequestFromStatefulSet(sts *appsv1.StatefulSet) (resource.Quantity, bool) {
	if len(sts.Spec.VolumeClaimTemplates) == 0 {
		return resource.Quantity{}, false
	}
	requests := sts.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests
	storage, ok := requests[corev1.ResourceStorage]
	return storage, ok
}

func recreateStatefulSetStorage(ctx context.Context, c client.Client, sts *appsv1.StatefulSet, claimName, appName string) error {
	if err := c.Delete(ctx, sts); err != nil {
		return err
	}

	return deletePVCsByPrefix(ctx, c, claimName, appName, sts.Namespace)
}

func resizePVCs(ctx context.Context, c client.Client, claimName, appName string, desired *appsv1.StatefulSet) error {
	desiredSize, hasDesired := storageRequestFromStatefulSet(desired)
	if !hasDesired {
		return nil
	}

	pvcs, err := listPVCsByPrefix(ctx, c, claimName, appName, desired.Namespace)
	if err != nil {
		return err
	}

	for _, pvc := range pvcs {
		currentSize, hasCurrent := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
		if !hasCurrent {
			continue
		}
		if currentSize.Cmp(desiredSize) >= 0 {
			continue
		}
		pvc.Spec.Resources.Requests[corev1.ResourceStorage] = desiredSize
		if err := c.Update(ctx, &pvc); err != nil {
			return err
		}
	}

	return nil
}

func deletePVCsByPrefix(ctx context.Context, c client.Client, claimName, appName, namespace string) error {
	pvcs, err := listPVCsByPrefix(ctx, c, claimName, appName, namespace)
	if err != nil {
		return err
	}

	for _, pvc := range pvcs {
		if err := c.Delete(ctx, &pvc); err != nil {
			return err
		}
	}

	return nil
}

func listPVCsByPrefix(ctx context.Context, c client.Client, claimName, appName, namespace string) ([]corev1.PersistentVolumeClaim, error) {
	pvcList := &corev1.PersistentVolumeClaimList{}
	if err := c.List(ctx, pvcList, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, err
	}

	prefix := fmt.Sprintf("%s-%s-", claimName, appName)
	filtered := make([]corev1.PersistentVolumeClaim, 0, len(pvcList.Items))
	for _, pvc := range pvcList.Items {
		if strings.HasPrefix(pvc.Name, prefix) {
			filtered = append(filtered, pvc)
		}
	}

	return filtered, nil
}
