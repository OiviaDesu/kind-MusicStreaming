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

// Package v1 chứa định nghĩa schema API cho nhóm API music v1
// +kubebuilder:object:generate=true
// +groupName=music.mixcorp.org
package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Hướng dẫn đọc nhanh:
// - Nếu chưa rõ schema của CRD, xem api/v1/musicservice_types.go.
// - Nếu chưa rõ nơi dùng CRD, xem internal/controller/musicservice_controller.go.

var (
	// GroupVersion là phiên bản nhóm dùng để đăng ký các đối tượng này
	GroupVersion = schema.GroupVersion{Group: "music.mixcorp.org", Version: "v1"}

	// SchemeBuilder dùng để thêm các kiểu Go vào scheme GroupVersionKind
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme thêm các kiểu trong group-version này vào scheme được cung cấp.
	AddToScheme = SchemeBuilder.AddToScheme
)
