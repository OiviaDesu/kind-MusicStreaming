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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Hướng dẫn đọc nhanh:
// - Nếu chưa rõ luồng reconcile, xem internal/controller/musicservice_controller.go.
// - Nếu chưa rõ cách tạo tài nguyên từ spec, xem internal/builder/resource_builder.go.
// - Nếu chưa rõ autoscaling/HPA, xem internal/reconciler/app.go.

// StreamingSpec định nghĩa cấu hình streaming
type StreamingSpec struct {
	// Bitrate cho streaming âm thanh (ví dụ: "320k", "192k")
	// +kubebuilder:validation:MinLength=1
	Bitrate string `json:"bitrate"`

	// MaxConnections là số kết nối đồng thời tối đa cho streaming
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10000
	MaxConnections int32 `json:"maxConnections"`
}

// StorageSpec định nghĩa yêu cầu lưu trữ
type StorageSpec struct {
	// Kích thước persistent volume (ví dụ: "10Gi", "100Gi")
	// +kubebuilder:validation:MinLength=1
	Size string `json:"size"`

	// UpdatePolicy kiểm soát cách áp dụng thay đổi kích thước lưu trữ
	// +kubebuilder:validation:Enum=Resize;Recreate
	// +optional
	UpdatePolicy StorageUpdatePolicy `json:"updatePolicy,omitempty"`
}

// StorageUpdatePolicy định nghĩa hành vi khi kích thước lưu trữ thay đổi
type StorageUpdatePolicy string

const (
	// StorageUpdatePolicyResize mở rộng PVC hiện có khi có thể
	StorageUpdatePolicyResize StorageUpdatePolicy = "Resize"
	// StorageUpdatePolicyRecreate xóa và tạo lại PVC cùng pod
	StorageUpdatePolicyRecreate StorageUpdatePolicy = "Recreate"
)

// AutoscalingSpec định nghĩa cấu hình autoscaling
type AutoscalingSpec struct {
	// MinReplicas là số replica tối thiểu
	// +kubebuilder:validation:Minimum=1
	MinReplicas int32 `json:"minReplicas"`

	// MaxReplicas là số replica tối đa
	// +kubebuilder:validation:Minimum=1
	MaxReplicas int32 `json:"maxReplicas"`

	// TargetCPUUtilizationPercentage là phần trăm sử dụng CPU mục tiêu
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	TargetCPUUtilizationPercentage int32 `json:"targetCPUUtilizationPercentage"`

	// TargetMemoryUtilizationPercentage là phần trăm sử dụng bộ nhớ mục tiêu
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +optional
	TargetMemoryUtilizationPercentage *int32 `json:"targetMemoryUtilizationPercentage,omitempty"`
}

// DatabaseSpec định nghĩa cấu hình cơ sở dữ liệu
type DatabaseSpec struct {
	// Enabled cho biết có triển khai cơ sở dữ liệu hay không
	Enabled bool `json:"enabled"`

	// Replicas là số lượng replica của cơ sở dữ liệu
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Image là image container của cơ sở dữ liệu
	// +optional
	Image string `json:"image,omitempty"`

	// Storage định nghĩa cấu hình lưu trữ của cơ sở dữ liệu
	// +optional
	Storage *StorageSpec `json:"storage,omitempty"`

	// RootPassword là mật khẩu root của cơ sở dữ liệu (nên dùng secret trong production)
	// +optional
	RootPassword string `json:"rootPassword,omitempty"`

	// Replication định nghĩa cấu hình replication giữa master và replica
	// +optional
	Replication *DatabaseReplicationSpec `json:"replication,omitempty"`

	// Autoscaling định nghĩa cấu hình autoscaling cho replica của cơ sở dữ liệu
	// +optional
	Autoscaling *AutoscalingSpec `json:"autoscaling,omitempty"`
}

// DatabaseReplicationSpec định nghĩa cấu hình replication
type DatabaseReplicationSpec struct {
	// Enabled bật/tắt replication (mặc định bật)
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// GTID bật/tắt GTID replication (mặc định bật)
	// +optional
	GTID *bool `json:"gtid,omitempty"`
}

// DatabaseStatus định nghĩa trạng thái quan sát được của cơ sở dữ liệu
type DatabaseStatus struct {
	// Phase biểu thị trạng thái hiện tại của cơ sở dữ liệu
	// +kubebuilder:validation:Enum=Pending;Progressing;Ready;Failed
	Phase string `json:"phase,omitempty"`

	// MasterReady cho biết master đã sẵn sàng hay chưa
	MasterReady bool `json:"masterReady,omitempty"`

	// ReplicasReady là số replica của cơ sở dữ liệu đã sẵn sàng
	ReplicasReady int32 `json:"replicasReady,omitempty"`

	// ReplicaEverCreated cho biết replica đã từng tồn tại hay chưa
	ReplicaEverCreated bool `json:"replicaEverCreated,omitempty"`

	// ReplicaLastSeen là thời điểm gần nhất quan sát thấy replica
	// +optional
	ReplicaLastSeen *metav1.Time `json:"replicaLastSeen,omitempty"`

	// ReplicaDeletionDetected cho biết replica đã bị xóa sau khi từng tồn tại
	ReplicaDeletionDetected bool `json:"replicaDeletionDetected,omitempty"`

	// ReplicationReady cho biết replication giữa master/replica đã sẵn sàng
	ReplicationReady bool `json:"replicationReady,omitempty"`
}

// MusicServiceSpec định nghĩa trạng thái mong muốn của MusicService
type MusicServiceSpec struct {
	// Replicas là số pod mong muốn
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	Replicas int32 `json:"replicas"`

	// Image là image container cần triển khai
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`

	// Port là cổng Service cho streaming nhạc
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`

	// Storage định nghĩa cấu hình lưu trữ
	Storage StorageSpec `json:"storage"`

	// Streaming định nghĩa cấu hình streaming
	Streaming StreamingSpec `json:"streaming"`

	// Resources định nghĩa tài nguyên tính toán cho container
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// Autoscaling định nghĩa cấu hình autoscaling
	// +optional
	Autoscaling *AutoscalingSpec `json:"autoscaling,omitempty"`

	// Database định nghĩa cấu hình cơ sở dữ liệu
	// +optional
	Database *DatabaseSpec `json:"database,omitempty"`
}

// MusicServiceStatus định nghĩa trạng thái quan sát được của MusicService
type MusicServiceStatus struct {
	// ObservedGeneration phản ánh generation mới nhất đã quan sát của MusicService
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// DesiredReplicas là số replica mong muốn trong spec
	DesiredReplicas int32 `json:"desiredReplicas,omitempty"`

	// ReadyReplicas là số pod đã sẵn sàng phục vụ lưu lượng
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// Phase biểu thị trạng thái hiện tại của MusicService (Pending, Progressing, Available, Failed)
	// +kubebuilder:validation:Enum=Pending;Progressing;Available;Degraded;Failed
	Phase string `json:"phase,omitempty"`

	// LastReconcileTime là thời điểm gần nhất tài nguyên được đồng bộ
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`

	// LastError là lỗi gần nhất trong quá trình đồng bộ
	LastError string `json:"lastError,omitempty"`

	// Conditions thể hiện các quan sát mới nhất về trạng thái của MusicService
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Database là trạng thái cơ sở dữ liệu nếu được bật
	// +optional
	Database *DatabaseStatus `json:"database,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.replicas"
// +kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyReplicas"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// MusicService là schema cho API musicservices
type MusicService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MusicServiceSpec   `json:"spec,omitempty"`
	Status MusicServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MusicServiceList chứa danh sách MusicService
type MusicServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MusicService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MusicService{}, &MusicServiceList{})
}
