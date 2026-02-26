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
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	musicv1 "github.com/example/managedapp-operator/api/v1"
	"github.com/example/managedapp-operator/internal/builder"
	"github.com/example/managedapp-operator/internal/tone"
)

// Hướng dẫn đọc nhanh:
// - Nếu chưa rõ các field autoscaling/replicas/resources, xem api/v1/musicservice_types.go.
// - Nếu chưa rõ cách tạo tài nguyên, xem internal/builder/resource_builder.go.
// - Nếu chưa rõ xử lý thay đổi dung lượng, xem internal/reconciler/storage.go.
// - Nếu chưa rõ luồng gọi, xem internal/controller/musicservice_controller.go.

// AppReconciler xử lý việc đồng bộ Service và StatefulSet của ứng dụng
type AppReconciler struct {
	client    client.Client
	builder   *builder.ResourceBuilder
	formatter *tone.Formatter
}

// NewAppReconciler tạo một reconciler mới cho ứng dụng
func NewAppReconciler(c client.Client, b *builder.ResourceBuilder, f *tone.Formatter) *AppReconciler {
	return &AppReconciler{
		client:    c,
		builder:   b,
		formatter: f,
	}
}

// ReconcileService đồng bộ Service của ứng dụng
func (ar *AppReconciler) ReconcileService(ctx context.Context, ms *musicv1.MusicService) error {
	log := log.FromContext(ctx)

	service := &corev1.Service{}
	serviceName := types.NamespacedName{Name: ms.Name, Namespace: ms.Namespace}

	err := ar.client.Get(ctx, serviceName, service)
	if err != nil && errors.IsNotFound(err) {
		service = ar.builder.BuildAppService(ms)
		log.Info("Creating new Service", "Service", ms.Name)
		return ar.client.Create(ctx, service)
	}

	return err
}

// ReconcileStatefulSet đồng bộ StatefulSet của ứng dụng
func (ar *AppReconciler) ReconcileStatefulSet(ctx context.Context, ms *musicv1.MusicService) error {
	log := log.FromContext(ctx)

	sts := &appsv1.StatefulSet{}
	stsName := types.NamespacedName{Name: ms.Name, Namespace: ms.Namespace}

	err := ar.client.Get(ctx, stsName, sts)
	if err != nil && errors.IsNotFound(err) {
		sts = ar.builder.BuildAppStatefulSet(ms)
		log.Info(ar.formatter.Format(ms, "Creating new StatefulSet"), "StatefulSet", ms.Name)
		return ar.client.Create(ctx, sts)
	} else if err != nil {
		return err
	}

	// Cập nhật nếu spec thay đổi
	desiredSts := ar.builder.BuildAppStatefulSet(ms)

	storageChanged := storageSizeChanged(sts, desiredSts)
	if storageChanged {
		policy := storageUpdatePolicy(ms.Spec.Storage)
		if policy == musicv1.StorageUpdatePolicyRecreate {
			log.Info("Recreating StatefulSet and PVCs due to storage size change", "StatefulSet", ms.Name)
			return recreateStatefulSetStorage(ctx, ar.client, sts, "music-data", ms.Name)
		}

		if err := resizePVCs(ctx, ar.client, "music-data", ms.Name, desiredSts); err != nil {
			return err
		}
	}

	if statefulSetNeedsUpdate(sts, desiredSts) {
		log.Info("Updating StatefulSet", "StatefulSet", ms.Name)
		sts.Spec = desiredSts.Spec
		return ar.client.Update(ctx, sts)
	}

	return nil
}

// ReconcileAutoscaler đồng bộ HorizontalPodAutoscaler
func (ar *AppReconciler) ReconcileAutoscaler(ctx context.Context, ms *musicv1.MusicService) error {
	log := log.FromContext(ctx)
	if ms.Spec.Autoscaling == nil {
		return ar.deleteAutoscalerIfExists(ctx, ms)
	}

	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	hpaName := types.NamespacedName{Name: ms.Name + "-autoscaler", Namespace: ms.Namespace}

	err := ar.client.Get(ctx, hpaName, hpa)
	if err != nil && errors.IsNotFound(err) {
		hpa = ar.builder.BuildAutoscaler(ms)
		log.Info("Creating new HorizontalPodAutoscaler", "HPA", hpaName.Name)
		return ar.client.Create(ctx, hpa)
	} else if err != nil {
		return err
	}

	desiredHpa := ar.builder.BuildAutoscaler(ms)
	if autoscalerNeedsUpdate(hpa, desiredHpa) {
		log.Info("Updating HorizontalPodAutoscaler", "HPA", hpaName.Name)
		hpa.Spec = desiredHpa.Spec
		return ar.client.Update(ctx, hpa)
	}

	return nil
}

// statefulSetNeedsUpdate kiểm tra xem spec của StatefulSet có cần cập nhật không
func statefulSetNeedsUpdate(current, desired *appsv1.StatefulSet) bool {
	if *current.Spec.Replicas != *desired.Spec.Replicas {
		return true
	}

	if !reflect.DeepEqual(current.Spec.Template.Spec.InitContainers, desired.Spec.Template.Spec.InitContainers) {
		return true
	}

	if !reflect.DeepEqual(current.Spec.Template.Spec.Volumes, desired.Spec.Template.Spec.Volumes) {
		return true
	}

	if len(current.Spec.Template.Spec.Containers) != len(desired.Spec.Template.Spec.Containers) {
		return true
	}

	for i := range current.Spec.Template.Spec.Containers {
		currentContainer := current.Spec.Template.Spec.Containers[i]
		desiredContainer := desired.Spec.Template.Spec.Containers[i]
		if currentContainer.Image != desiredContainer.Image {
			return true
		}
		if !reflect.DeepEqual(currentContainer.Resources, desiredContainer.Resources) {
			return true
		}
		if !reflect.DeepEqual(currentContainer.Env, desiredContainer.Env) {
			return true
		}
		if !reflect.DeepEqual(currentContainer.VolumeMounts, desiredContainer.VolumeMounts) {
			return true
		}
		if !reflect.DeepEqual(currentContainer.Ports, desiredContainer.Ports) {
			return true
		}
		if !reflect.DeepEqual(currentContainer.ReadinessProbe, desiredContainer.ReadinessProbe) {
			return true
		}
		if !reflect.DeepEqual(currentContainer.LivenessProbe, desiredContainer.LivenessProbe) {
			return true
		}
	}

	return false
}

func autoscalerNeedsUpdate(current, desired *autoscalingv2.HorizontalPodAutoscaler) bool {
	if current.Spec.MaxReplicas != desired.Spec.MaxReplicas {
		return true
	}
	if current.Spec.MinReplicas == nil || desired.Spec.MinReplicas == nil {
		return current.Spec.MinReplicas != desired.Spec.MinReplicas
	}
	if *current.Spec.MinReplicas != *desired.Spec.MinReplicas {
		return true
	}

	if len(current.Spec.Metrics) != len(desired.Spec.Metrics) {
		return true
	}

	for i, metric := range current.Spec.Metrics {
		desiredMetric := desired.Spec.Metrics[i]
		if metric.Type != desiredMetric.Type {
			return true
		}
		if metric.Resource == nil || desiredMetric.Resource == nil {
			return metric.Resource != desiredMetric.Resource
		}
		if metric.Resource.Name != desiredMetric.Resource.Name {
			return true
		}
		if metric.Resource.Target.AverageUtilization == nil || desiredMetric.Resource.Target.AverageUtilization == nil {
			return metric.Resource.Target.AverageUtilization != desiredMetric.Resource.Target.AverageUtilization
		}
		if *metric.Resource.Target.AverageUtilization != *desiredMetric.Resource.Target.AverageUtilization {
			return true
		}
	}

	return false
}

func (ar *AppReconciler) deleteAutoscalerIfExists(ctx context.Context, ms *musicv1.MusicService) error {
	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	hpaName := types.NamespacedName{Name: ms.Name + "-autoscaler", Namespace: ms.Namespace}

	err := ar.client.Get(ctx, hpaName, hpa)
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	return ar.client.Delete(ctx, hpa)
}
