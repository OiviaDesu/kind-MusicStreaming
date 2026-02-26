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
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	musicv1 "github.com/example/managedapp-operator/api/v1"
)

// Quick navigation for understanding the builder:
// - For field specifications, see api/v1/musicservice_types.go
// - For builder usage, see internal/reconciler/app.go and database.go
// - For overall flow, see internal/controller/musicservice_controller.go

// ResourceBuilder constructs Kubernetes resources from MusicService specifications
type ResourceBuilder struct {
	scheme *runtime.Scheme
}

// NewResourceBuilder tạo một ResourceBuilder mới
func NewResourceBuilder(scheme *runtime.Scheme) *ResourceBuilder {
	return &ResourceBuilder{
		scheme: scheme,
	}
}

// BuildAppService xây dựng Service cho ứng dụng
func (b *ResourceBuilder) BuildAppService(ms *musicv1.MusicService) *corev1.Service {
	labels := b.getLabels(ms, "app")

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ms.Name,
			Namespace: ms.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ms, musicv1.GroupVersion.WithKind("MusicService")),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":       ms.Name,
				"component": "music-service",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       ms.Spec.Port,
					TargetPort: intstr.FromInt(80),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}

// BuildAppStatefulSet xây dựng StatefulSet cho ứng dụng
func (b *ResourceBuilder) BuildAppStatefulSet(ms *musicv1.MusicService) *appsv1.StatefulSet {
	labels := b.getLabels(ms, "app")
	podLabels := map[string]string{
		"app":       ms.Name,
		"component": "music-service",
	}

	resources := corev1.ResourceRequirements{}
	if ms.Spec.Resources != nil {
		resources = *ms.Spec.Resources
	}

	storageSize := resource.MustParse(ms.Spec.Storage.Size)

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ms.Name,
			Namespace: ms.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ms, musicv1.GroupVersion.WithKind("MusicService")),
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &ms.Spec.Replicas,
			ServiceName: ms.Name,
			Selector: &metav1.LabelSelector{
				MatchLabels: podLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:      "music-service",
							Image:     ms.Spec.Image,
							Resources: resources,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 80,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "STREAMING_BITRATE",
									Value: ms.Spec.Streaming.Bitrate,
								},
								{
									Name:  "MAX_CONNECTIONS",
									Value: fmt.Sprintf("%d", ms.Spec.Streaming.MaxConnections),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "music-data",
									MountPath: "/data",
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "music-data",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: storageSize,
							},
						},
					},
				},
			},
		},
	}
}

// BuildDatabaseMasterStatefulSet xây dựng StatefulSet master của cơ sở dữ liệu
func (b *ResourceBuilder) BuildDatabaseMasterStatefulSet(ms *musicv1.MusicService) *appsv1.StatefulSet {
	labels := b.getLabels(ms, "db-master")
	podLabels := map[string]string{
		"app":       ms.Name,
		"component": "db-master",
	}

	config := buildDatabaseConfig(ms)
	replicas := int32(1)

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ms.Name + "-db-master",
			Namespace: ms.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ms, musicv1.GroupVersion.WithKind("MusicService")),
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: ms.Name + "-db-master",
			Selector: &metav1.LabelSelector{
				MatchLabels: podLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:    "init-db-config",
							Image:   config.image,
							Command: []string{"/bin/sh", "-c", buildMasterConfigScript()},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "db-config",
									MountPath: "/db-config",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "mariadb",
							Image: config.image,
							Env: []corev1.EnvVar{
								{
									Name:  "MYSQL_ROOT_PASSWORD",
									Value: config.rootPassword,
								},
								{
									Name:  "MYSQL_DATABASE",
									Value: "musicdb",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "mysql",
									ContainerPort: 3306,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"/bin/sh", "-c", "mysqladmin ping -uroot -p$MYSQL_ROOT_PASSWORD"},
									},
								},
								InitialDelaySeconds: 10,
								PeriodSeconds:       10,
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"/bin/sh", "-c", "mysqladmin ping -uroot -p$MYSQL_ROOT_PASSWORD"},
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       20,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "db-data",
									MountPath: "/var/lib/mysql",
								},
								{
									Name:      "db-config",
									MountPath: "/etc/mysql/conf.d",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "db-config",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "db-data",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: config.storageSize,
							},
						},
					},
				},
			},
		},
	}
}

// BuildDatabaseReplicaStatefulSet xây dựng StatefulSet replica của cơ sở dữ liệu
func (b *ResourceBuilder) BuildDatabaseReplicaStatefulSet(ms *musicv1.MusicService) *appsv1.StatefulSet {
	labels := b.getLabels(ms, "db-replica")
	podLabels := map[string]string{
		"app":       ms.Name,
		"component": "db-replica",
	}

	config := buildDatabaseConfig(ms)
	replicationSetupScript := buildReplicaSetupScript(config.masterHost)
	initContainers := []corev1.Container{
		{
			Name:    "init-db-config",
			Image:   config.image,
			Command: []string{"/bin/sh", "-c", buildReplicaConfigScript()},
			Env: []corev1.EnvVar{
				{
					Name: "POD_NAME",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
					},
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "db-config",
					MountPath: "/db-config",
				},
			},
		},
	}
	volumes := []corev1.Volume{
		{
			Name: "db-config",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
	replicaEnv := []corev1.EnvVar{
		{
			Name:  "MYSQL_ROOT_PASSWORD",
			Value: config.rootPassword,
		},
		{
			Name:  "MYSQL_DATABASE",
			Value: "musicdb",
		},
	}
	replicaVolumeMounts := []corev1.VolumeMount{
		{
			Name:      "db-data",
			MountPath: "/var/lib/mysql",
		},
		{
			Name:      "db-config",
			MountPath: "/etc/mysql/conf.d",
		},
	}

	if config.replicationEnabled {
		replicaEnv = append(replicaEnv,
			corev1.EnvVar{
				Name: "REPLICATION_USER",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: config.replicationSecret,
						},
						Key: "username",
					},
				},
			},
			corev1.EnvVar{
				Name: "REPLICATION_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: config.replicationSecret,
						},
						Key: "password",
					},
				},
			},
		)
	}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ms.Name + "-db-replica",
			Namespace: ms.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ms, musicv1.GroupVersion.WithKind("MusicService")),
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &config.replicas,
			ServiceName: ms.Name + "-db-replica",
			Selector: &metav1.LabelSelector{
				MatchLabels: podLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: corev1.PodSpec{
					InitContainers: initContainers,
					Containers: append([]corev1.Container{
						{
							Name:  "mariadb",
							Image: config.image,
							Env:   replicaEnv,
							Ports: []corev1.ContainerPort{
								{
									Name:          "mysql",
									ContainerPort: 3306,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"/bin/sh", "-c", "mysqladmin ping -uroot -p$MYSQL_ROOT_PASSWORD"},
									},
								},
								InitialDelaySeconds: 10,
								PeriodSeconds:       10,
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"/bin/sh", "-c", "mysqladmin ping -uroot -p$MYSQL_ROOT_PASSWORD"},
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       20,
							},
							VolumeMounts: replicaVolumeMounts,
						},
					},
						buildReplicaSetupContainer(config, replicationSetupScript)...),
					Volumes: volumes,
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "db-data",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: config.storageSize,
							},
						},
					},
				},
			},
		},
	}
}

// BuildDatabaseMasterService xây dựng Service master của cơ sở dữ liệu
func (b *ResourceBuilder) BuildDatabaseMasterService(ms *musicv1.MusicService) *corev1.Service {
	labels := b.getLabels(ms, "db-master")

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ms.Name + "-db-master",
			Namespace: ms.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ms, musicv1.GroupVersion.WithKind("MusicService")),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":       ms.Name,
				"component": "db-master",
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "mysql",
					Port:     3306,
					Protocol: corev1.ProtocolTCP,
				},
			},
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "None",
		},
	}
}

// BuildDatabaseReadService xây dựng Service đọc của cơ sở dữ liệu
func (b *ResourceBuilder) BuildDatabaseReadService(ms *musicv1.MusicService) *corev1.Service {
	labels := b.getLabels(ms, "db-read")

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ms.Name + "-db-read",
			Namespace: ms.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ms, musicv1.GroupVersion.WithKind("MusicService")),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":       ms.Name,
				"component": "db-replica",
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "mysql",
					Port:     3306,
					Protocol: corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}

// BuildAutoscaler xây dựng HorizontalPodAutoscaler cho StatefulSet của ứng dụng
func (b *ResourceBuilder) BuildAutoscaler(ms *musicv1.MusicService) *autoscalingv2.HorizontalPodAutoscaler {
	labels := b.getLabels(ms, "autoscaler")
	metrics := []autoscalingv2.MetricSpec{
		buildResourceMetric(corev1.ResourceCPU, ms.Spec.Autoscaling.TargetCPUUtilizationPercentage),
	}

	if ms.Spec.Autoscaling.TargetMemoryUtilizationPercentage != nil {
		metrics = append(metrics, buildResourceMetric(corev1.ResourceMemory, *ms.Spec.Autoscaling.TargetMemoryUtilizationPercentage))
	}

	return &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ms.Name + "-autoscaler",
			Namespace: ms.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ms, musicv1.GroupVersion.WithKind("MusicService")),
			},
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "StatefulSet",
				Name:       ms.Name,
			},
			MinReplicas: &ms.Spec.Autoscaling.MinReplicas,
			MaxReplicas: ms.Spec.Autoscaling.MaxReplicas,
			Metrics:     metrics,
		},
	}
}

// BuildDatabaseReplicaAutoscaler xây dựng HorizontalPodAutoscaler cho StatefulSet replica của cơ sở dữ liệu
func (b *ResourceBuilder) BuildDatabaseReplicaAutoscaler(ms *musicv1.MusicService) *autoscalingv2.HorizontalPodAutoscaler {
	labels := b.getLabels(ms, "db-autoscaler")
	autoscaling := ms.Spec.Database.Autoscaling
	metrics := []autoscalingv2.MetricSpec{
		buildResourceMetric(corev1.ResourceCPU, autoscaling.TargetCPUUtilizationPercentage),
	}

	if autoscaling.TargetMemoryUtilizationPercentage != nil {
		metrics = append(metrics, buildResourceMetric(corev1.ResourceMemory, *autoscaling.TargetMemoryUtilizationPercentage))
	}

	return &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ms.Name + "-db-replica-autoscaler",
			Namespace: ms.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ms, musicv1.GroupVersion.WithKind("MusicService")),
			},
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "StatefulSet",
				Name:       ms.Name + "-db-replica",
			},
			MinReplicas: &autoscaling.MinReplicas,
			MaxReplicas: autoscaling.MaxReplicas,
			Metrics:     metrics,
		},
	}
}

// Helper functions for building labels and metrics

func (b *ResourceBuilder) getLabels(ms *musicv1.MusicService, component string) map[string]string {
	labels := map[string]string{
		"app":                          ms.Name,
		"component":                    component,
		"app.kubernetes.io/name":       "music-service",
		"app.kubernetes.io/instance":   ms.Name,
		"app.kubernetes.io/managed-by": "music-operator",
	}

	return labels
}

func buildResourceMetric(resourceName corev1.ResourceName, targetUtilization int32) autoscalingv2.MetricSpec {
	return autoscalingv2.MetricSpec{
		Type: autoscalingv2.ResourceMetricSourceType,
		Resource: &autoscalingv2.ResourceMetricSource{
			Name: resourceName,
			Target: autoscalingv2.MetricTarget{
				Type:               autoscalingv2.UtilizationMetricType,
				AverageUtilization: &targetUtilization,
			},
		},
	}
}

type databaseConfig struct {
	image              string
	storageSize        resource.Quantity
	rootPassword       string
	replicas           int32
	masterHost         string
	replicationEnabled bool
	replicationGTID    bool
	replicationSecret  string
}

func buildDatabaseConfig(ms *musicv1.MusicService) databaseConfig {
	config := databaseConfig{
		image:              "mariadb:10.11",
		storageSize:        resource.MustParse("10Gi"),
		rootPassword:       "rootpass",
		replicas:           0,
		masterHost:         ms.Name + "-db-master",
		replicationEnabled: true,
		replicationGTID:    true,
		replicationSecret:  replicationSecretName(ms),
	}

	if ms.Spec.Database == nil {
		return config
	}

	config.replicas = ms.Spec.Database.Replicas
	if ms.Spec.Database.Image != "" {
		config.image = ms.Spec.Database.Image
	}
	if ms.Spec.Database.Storage != nil {
		config.storageSize = resource.MustParse(ms.Spec.Database.Storage.Size)
	}
	if ms.Spec.Database.RootPassword != "" {
		config.rootPassword = ms.Spec.Database.RootPassword
	}
	if ms.Spec.Database.Replication != nil {
		if ms.Spec.Database.Replication.Enabled != nil {
			config.replicationEnabled = *ms.Spec.Database.Replication.Enabled
		}
		if ms.Spec.Database.Replication.GTID != nil {
			config.replicationGTID = *ms.Spec.Database.Replication.GTID
		}
	}

	return config
}

func replicationSecretName(ms *musicv1.MusicService) string {
	return ms.Name + "-db-replication"
}

func buildReplicaSetupScript(masterHost string) string {
	return fmt.Sprintf(`
#!/bin/bash
set -e
echo "Waiting for local MariaDB to be ready..."
until mysql -h 127.0.0.1 -P 3306 -uroot -p${MYSQL_ROOT_PASSWORD} -e "SELECT 1" > /dev/null 2>&1; do
	sleep 2
done
echo "Waiting for master to be ready..."
until mysql -h %[1]s -P 3306 -uroot -p${MYSQL_ROOT_PASSWORD} -e "SELECT 1" > /dev/null 2>&1; do
	sleep 2
done
echo "Master is ready, ensuring replication user..."
mysql -h %[1]s -P 3306 -uroot -p${MYSQL_ROOT_PASSWORD} -e "CREATE USER IF NOT EXISTS '${REPLICATION_USER}'@'%%' IDENTIFIED BY '${REPLICATION_PASSWORD}'; GRANT REPLICATION SLAVE ON *.* TO '${REPLICATION_USER}'@'%%'; FLUSH PRIVILEGES;"
echo "Configuring replica..."
mysql -h 127.0.0.1 -P 3306 -uroot -p${MYSQL_ROOT_PASSWORD} -e "STOP SLAVE; RESET SLAVE ALL; CHANGE MASTER TO MASTER_HOST='%[1]s', MASTER_USER='${REPLICATION_USER}', MASTER_PASSWORD='${REPLICATION_PASSWORD}', MASTER_PORT=3306, MASTER_USE_GTID=slave_pos; START SLAVE;"
mysql -h 127.0.0.1 -P 3306 -uroot -p${MYSQL_ROOT_PASSWORD} -e "SHOW SLAVE STATUS\\G" | grep -E "Slave_IO_Running: Yes|Slave_SQL_Running: Yes" || true
echo "Replication setup complete. Sleeping..."
sleep infinity
`, masterHost)
}

func buildMasterConfigScript() string {
	return `
set -e
cat <<'EOF' > /db-config/server-id.cnf
[mysqld]
server-id=1
log_bin=mysql-bin
binlog_format=ROW
gtid_strict_mode=ON
log_slave_updates=ON
EOF
`
}

func buildReplicaConfigScript() string {
	return `
set -e
ordinal=${POD_NAME##*-}
server_id=$((200 + ordinal))
cat <<EOF > /db-config/server-id.cnf
[mysqld]
server-id=${server_id}
log_bin=mysql-bin
binlog_format=ROW
gtid_strict_mode=ON
log_slave_updates=ON
read_only=ON
skip_slave_start=1
EOF
`
}

func buildReplicaSetupContainer(config databaseConfig, script string) []corev1.Container {
	if !config.replicationEnabled {
		return nil
	}

	return []corev1.Container{
		{
			Name:    "replication-setup",
			Image:   config.image,
			Command: []string{"/bin/sh", "-c", script},
			Env: []corev1.EnvVar{
				{
					Name:  "MYSQL_ROOT_PASSWORD",
					Value: config.rootPassword,
				},
				{
					Name: "REPLICATION_USER",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: config.replicationSecret,
							},
							Key: "username",
						},
					},
				},
				{
					Name: "REPLICATION_PASSWORD",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: config.replicationSecret,
							},
							Key: "password",
						},
					},
				},
			},
		},
	}
}
