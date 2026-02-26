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
	"crypto/rand"
	"encoding/hex"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	musicv1 "github.com/example/managedapp-operator/api/v1"
	"github.com/example/managedapp-operator/internal/builder"
	"github.com/example/managedapp-operator/internal/tone"
)

// DatabaseReconciler handles reconciliation of database StatefulSets and Services
type DatabaseReconciler struct {
	client    client.Client
	builder   *builder.ResourceBuilder
	formatter *tone.Formatter
}

// NewDatabaseReconciler creates a new database reconciler
func NewDatabaseReconciler(c client.Client, b *builder.ResourceBuilder, f *tone.Formatter) *DatabaseReconciler {
	return &DatabaseReconciler{
		client:    c,
		builder:   b,
		formatter: f,
	}
}

// ReconcileMaster reconciles the database master StatefulSet
func (dr *DatabaseReconciler) ReconcileMaster(ctx context.Context, ms *musicv1.MusicService) error {
	log := log.FromContext(ctx)

	sts := &appsv1.StatefulSet{}
	stsName := types.NamespacedName{
		Name:      ms.Name + "-db-master",
		Namespace: ms.Namespace,
	}

	err := dr.client.Get(ctx, stsName, sts)
	if err != nil && errors.IsNotFound(err) {
		sts = dr.builder.BuildDatabaseMasterStatefulSet(ms)
		log.Info(dr.formatter.Format(ms, "Creating DB Master"), "StatefulSet", stsName.Name)
		return dr.client.Create(ctx, sts)
	}
	if err != nil {
		return err
	}

	desiredSts := dr.builder.BuildDatabaseMasterStatefulSet(ms)
	storageChanged := storageSizeChanged(sts, desiredSts)
	if storageChanged {
		policy := storageUpdatePolicy(databaseStorageSpec(ms))
		if policy == musicv1.StorageUpdatePolicyRecreate {
			log.Info("Recreating DB master StatefulSet and PVCs due to storage size change", "StatefulSet", stsName.Name)
			return recreateStatefulSetStorage(ctx, dr.client, sts, "db-data", ms.Name+"-db-master")
		}
		if err := resizePVCs(ctx, dr.client, "db-data", ms.Name+"-db-master", desiredSts); err != nil {
			return err
		}
	}

	if statefulSetNeedsUpdate(sts, desiredSts) {
		log.Info("Updating DB master StatefulSet", "StatefulSet", stsName.Name)
		sts.Spec = desiredSts.Spec
		return dr.client.Update(ctx, sts)
	}

	return nil
}

// ReconcileReplicas reconciles the database replica StatefulSet
func (dr *DatabaseReconciler) ReconcileReplicas(ctx context.Context, ms *musicv1.MusicService) error {
	if ms.Spec.Database.Replicas == 0 {
		return nil
	}

	if _, err := dr.ensureReplicationSecret(ctx, ms); err != nil {
		return err
	}

	log := log.FromContext(ctx)

	sts := &appsv1.StatefulSet{}
	stsName := types.NamespacedName{
		Name:      ms.Name + "-db-replica",
		Namespace: ms.Namespace,
	}

	err := dr.client.Get(ctx, stsName, sts)
	if err != nil && errors.IsNotFound(err) {
		sts = dr.builder.BuildDatabaseReplicaStatefulSet(ms)
		log.Info(dr.formatter.Format(ms, "Creating DB Replicas"), "StatefulSet", stsName.Name)
		return dr.client.Create(ctx, sts)
	}
	if err != nil {
		return err
	}

	desiredSts := dr.builder.BuildDatabaseReplicaStatefulSet(ms)
	storageChanged := storageSizeChanged(sts, desiredSts)
	if storageChanged {
		policy := storageUpdatePolicy(databaseStorageSpec(ms))
		if policy == musicv1.StorageUpdatePolicyRecreate {
			log.Info("Recreating DB replica StatefulSet and PVCs due to storage size change", "StatefulSet", stsName.Name)
			return recreateStatefulSetStorage(ctx, dr.client, sts, "db-data", ms.Name+"-db-replica")
		}
		if err := resizePVCs(ctx, dr.client, "db-data", ms.Name+"-db-replica", desiredSts); err != nil {
			return err
		}
	}

	if statefulSetNeedsUpdate(sts, desiredSts) {
		log.Info("Updating DB replica StatefulSet", "StatefulSet", stsName.Name)
		sts.Spec = desiredSts.Spec
		return dr.client.Update(ctx, sts)
	}

	return nil
}

// ReconcileServices reconciles the database Services
func (dr *DatabaseReconciler) ReconcileServices(ctx context.Context, ms *musicv1.MusicService) error {
	masterSvc := &corev1.Service{}
	masterSvcName := types.NamespacedName{
		Name:      ms.Name + "-db-master",
		Namespace: ms.Namespace,
	}

	err := dr.client.Get(ctx, masterSvcName, masterSvc)
	if err != nil && errors.IsNotFound(err) {
		masterSvc = dr.builder.BuildDatabaseMasterService(ms)
		if err := dr.client.Create(ctx, masterSvc); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Service đọc (dành cho replica)
	if ms.Spec.Database.Replicas > 0 {
		readSvc := &corev1.Service{}
		readSvcName := types.NamespacedName{
			Name:      ms.Name + "-db-read",
			Namespace: ms.Namespace,
		}

		err := dr.client.Get(ctx, readSvcName, readSvc)
		if err != nil && errors.IsNotFound(err) {
			readSvc = dr.builder.BuildDatabaseReadService(ms)
			return dr.client.Create(ctx, readSvc)
		}
	}

	return nil
}

// ReconcileAutoscaler reconciles the HPA for database replicas
func (dr *DatabaseReconciler) ReconcileAutoscaler(ctx context.Context, ms *musicv1.MusicService) error {
	if ms.Spec.Database.Autoscaling == nil || ms.Spec.Database.Replicas == 0 {
		return nil
	}

	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	hpaName := types.NamespacedName{
		Name:      ms.Name + "-db-replica-autoscaler",
		Namespace: ms.Namespace,
	}

	err := dr.client.Get(ctx, hpaName, hpa)
	if err != nil && errors.IsNotFound(err) {
		hpa = dr.builder.BuildDatabaseReplicaAutoscaler(ms)
		return dr.client.Create(ctx, hpa)
	}
	if err != nil {
		return err
	}

	desired := dr.builder.BuildDatabaseReplicaAutoscaler(ms)
	if !reflect.DeepEqual(hpa.Spec, desired.Spec) {
		hpa.Spec = desired.Spec
		return dr.client.Update(ctx, hpa)
	}

	return nil
}

func databaseStorageSpec(ms *musicv1.MusicService) musicv1.StorageSpec {
	if ms.Spec.Database != nil && ms.Spec.Database.Storage != nil {
		return *ms.Spec.Database.Storage
	}
	return musicv1.StorageSpec{}
}

func (dr *DatabaseReconciler) ensureReplicationSecret(ctx context.Context, ms *musicv1.MusicService) (*corev1.Secret, error) {
	if !replicationEnabled(ms) || ms.Spec.Database.Replicas == 0 {
		return nil, nil
	}

	secretName := types.NamespacedName{
		Name:      ms.Name + "-db-replication",
		Namespace: ms.Namespace,
	}
	secret := &corev1.Secret{}
	if err := dr.client.Get(ctx, secretName, secret); err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}

		username := []byte("repl")
		password, err := generatePassword(16)
		if err != nil {
			return nil, err
		}

		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName.Name,
				Namespace: secretName.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(ms, musicv1.GroupVersion.WithKind("MusicService")),
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"username": username,
				"password": []byte(password),
			},
		}

		return secret, dr.client.Create(ctx, secret)
	}

	updated := false
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}
	if _, ok := secret.Data["username"]; !ok {
		secret.Data["username"] = []byte("repl")
		updated = true
	}
	if _, ok := secret.Data["password"]; !ok {
		password, err := generatePassword(16)
		if err != nil {
			return nil, err
		}
		secret.Data["password"] = []byte(password)
		updated = true
	}
	if updated {
		if err := dr.client.Update(ctx, secret); err != nil {
			return nil, err
		}
	}

	return secret, nil
}

func generatePassword(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func replicationEnabled(ms *musicv1.MusicService) bool {
	if ms.Spec.Database == nil {
		return false
	}
	if ms.Spec.Database.Replication == nil || ms.Spec.Database.Replication.Enabled == nil {
		return true
	}
	return *ms.Spec.Database.Replication.Enabled
}
