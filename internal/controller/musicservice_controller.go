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
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	musicv1 "github.com/example/managedapp-operator/api/v1"
	"github.com/example/managedapp-operator/internal/builder"
	"github.com/example/managedapp-operator/internal/reconciler"
	"github.com/example/managedapp-operator/internal/status"
	"github.com/example/managedapp-operator/internal/tone"
)

const (
	musicServiceFinalizerName = "music.mixcorp.org/finalizer"
)

// MusicServiceReconciler reconciles a MusicService object
type MusicServiceReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	// Dependencies are injected by the manager
	resourceBuilder    *builder.ResourceBuilder
	statusManager      *status.Manager
	appReconciler      *reconciler.AppReconciler
	databaseReconciler *reconciler.DatabaseReconciler
	messageFormatter   *tone.Formatter
}

// +kubebuilder:rbac:groups=music.mixcorp.org,resources=musicservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=music.mixcorp.org,resources=musicservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=music.mixcorp.org,resources=musicservices/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete

// Reconcile implements the reconciliation loop for MusicService
func (r *MusicServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the MusicService object
	musicService := &musicv1.MusicService{}
	if err := r.Get(ctx, req.NamespacedName, musicService); err != nil {
		if errors.IsNotFound(err) {
			log.Info("MusicService resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to get MusicService")
		return ctrl.Result{}, err
	}

	log.Info(r.messageFormatter.Format(musicService, "Reconciling MusicService"), "MusicService", musicService.Name)
	r.Recorder.Event(musicService, corev1.EventTypeNormal, "Reconciling", r.messageFormatter.Format(musicService, "Starting reconciliation"))

	// Handle deletion with finalizer
	if musicService.ObjectMeta.DeletionTimestamp != nil {
		if controllerutil.ContainsFinalizer(musicService, musicServiceFinalizerName) {
			log.Info(r.messageFormatter.Format(musicService, "Deleting associated resources"), "MusicService", musicService.Name)
			r.Recorder.Event(musicService, corev1.EventTypeNormal, "Deleting", r.messageFormatter.Format(musicService, "Cleaning up resources"))

			controllerutil.RemoveFinalizer(musicService, musicServiceFinalizerName)
			if err := r.Update(ctx, musicService); err != nil {
				log.Error(err, "failed to remove finalizer")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not already present
	if !controllerutil.ContainsFinalizer(musicService, musicServiceFinalizerName) {
		controllerutil.AddFinalizer(musicService, musicServiceFinalizerName)
		if err := r.Update(ctx, musicService); err != nil {
			log.Error(err, "failed to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// Initialize status
	musicService.Status.ObservedGeneration = musicService.Generation
	musicService.Status.DesiredReplicas = musicService.Spec.Replicas

	// Reconcile application service
	if err := r.appReconciler.ReconcileService(ctx, musicService); err != nil {
		return ctrl.Result{}, r.statusManager.UpdateError(ctx, musicService, "ServiceFailed", err.Error())
	}

	// Reconcile application StatefulSet
	if err := r.appReconciler.ReconcileStatefulSet(ctx, musicService); err != nil {
		return ctrl.Result{}, r.statusManager.UpdateError(ctx, musicService, "StatefulSetFailed", err.Error())
	}

	// Reconcile autoscaler if configured
	if err := r.appReconciler.ReconcileAutoscaler(ctx, musicService); err != nil {
		return ctrl.Result{}, r.statusManager.UpdateError(ctx, musicService, "AutoscalerFailed", err.Error())
	}

	// Reconcile database if enabled
	if databaseEnabled(musicService) {
		if musicService.Status.Database == nil {
			musicService.Status.Database = &musicv1.DatabaseStatus{}
		}

		if databaseHAEnabled(musicService) {
			// Chế độ Galera Cluster: tất cả node ngang hàng, không gián đoạn khi master chết
			if err := r.databaseReconciler.ReconcileGalera(ctx, musicService); err != nil {
				return ctrl.Result{}, r.statusManager.UpdateError(ctx, musicService, "DBGaleraFailed", err.Error())
			}
			if err := r.databaseReconciler.ReconcileGaleraServices(ctx, musicService); err != nil {
				return ctrl.Result{}, r.statusManager.UpdateError(ctx, musicService, "DBGaleraServicesFailed", err.Error())
			}
		} else {
			// Chế độ master/replica truyền thống
			if err := r.databaseReconciler.ReconcileMaster(ctx, musicService); err != nil {
				return ctrl.Result{}, r.statusManager.UpdateError(ctx, musicService, "DBMasterFailed", err.Error())
			}

			if err := r.databaseReconciler.ReconcileReplicas(ctx, musicService); err != nil {
				return ctrl.Result{}, r.statusManager.UpdateError(ctx, musicService, "DBReplicasFailed", err.Error())
			}

			if err := r.databaseReconciler.ReconcileServices(ctx, musicService); err != nil {
				return ctrl.Result{}, r.statusManager.UpdateError(ctx, musicService, "DBServicesFailed", err.Error())
			}
		}

		if err := r.databaseReconciler.ReconcileAutoscaler(ctx, musicService); err != nil {
			return ctrl.Result{}, r.statusManager.UpdateError(ctx, musicService, "DBAutoscalerFailed", err.Error())
		}
	}

	// Sync status from StatefulSet
	appSts := &appsv1.StatefulSet{}
	appStsName := types.NamespacedName{Name: musicService.Name, Namespace: musicService.Namespace}
	if err := r.Get(ctx, appStsName, appSts); err == nil {
		if err := r.statusManager.UpdateFromAppStatefulSet(ctx, musicService, appSts); err != nil {
			log.Error(err, "failed to update app statefulset status")
			return ctrl.Result{}, err
		}
		r.Recorder.Event(musicService, corev1.EventTypeNormal, "Ready", r.messageFormatter.Format(musicService, "Service is ready"))
	}

	// Update database status if enabled
	if databaseEnabled(musicService) {
		if err := r.statusManager.UpdateDatabase(ctx, musicService); err != nil {
			log.Error(err, "failed to update database status")
			return ctrl.Result{}, err
		}
	}

	// Mark reconciliation as complete
	if err := r.statusManager.UpdateReconciled(ctx, musicService); err != nil {
		log.Error(err, "failed to update MusicService status")
		return ctrl.Result{}, err
	}

	// Requeue if not all replicas are ready
	if musicService.Status.ReadyReplicas < musicService.Spec.Replicas {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MusicServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Set up event recorder
	r.Recorder = mgr.GetEventRecorderFor("musicservice-controller")

	// Initialize dependencies
	r.resourceBuilder = builder.NewResourceBuilder(r.Scheme)
	r.statusManager = status.NewManager(r.Client)
	r.messageFormatter = tone.NewFormatter()
	r.appReconciler = reconciler.NewAppReconciler(r.Client, r.resourceBuilder, r.messageFormatter)
	r.databaseReconciler = reconciler.NewDatabaseReconciler(r.Client, r.resourceBuilder, r.messageFormatter)

	return ctrl.NewControllerManagedBy(mgr).
		For(&musicv1.MusicService{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

func databaseEnabled(ms *musicv1.MusicService) bool {
	return ms.Spec.Database != nil && ms.Spec.Database.Enabled
}

func databaseHAEnabled(ms *musicv1.MusicService) bool {
	return ms.Spec.Database != nil &&
		ms.Spec.Database.HighAvailability != nil &&
		ms.Spec.Database.HighAvailability.Enabled
}
