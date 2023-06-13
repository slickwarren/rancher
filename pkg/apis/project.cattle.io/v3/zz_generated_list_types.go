/*
Copyright 2023 Rancher Labs, Inc.

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

// Code generated by main. DO NOT EDIT.

// +k8s:deepcopy-gen=package
// +groupName=project.cattle.io
package v3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AppList is a list of App resources
type AppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []App `json:"items"`
}

func NewApp(namespace, name string, obj App) *App {
	obj.APIVersion, obj.Kind = SchemeGroupVersion.WithKind("App").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	return &obj
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AppRevisionList is a list of AppRevision resources
type AppRevisionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AppRevision `json:"items"`
}

func NewAppRevision(namespace, name string, obj AppRevision) *AppRevision {
	obj.APIVersion, obj.Kind = SchemeGroupVersion.WithKind("AppRevision").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	return &obj
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BasicAuthList is a list of BasicAuth resources
type BasicAuthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []BasicAuth `json:"items"`
}

func NewBasicAuth(namespace, name string, obj BasicAuth) *BasicAuth {
	obj.APIVersion, obj.Kind = SchemeGroupVersion.WithKind("BasicAuth").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	return &obj
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CertificateList is a list of Certificate resources
type CertificateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Certificate `json:"items"`
}

func NewCertificate(namespace, name string, obj Certificate) *Certificate {
	obj.APIVersion, obj.Kind = SchemeGroupVersion.WithKind("Certificate").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	return &obj
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DockerCredentialList is a list of DockerCredential resources
type DockerCredentialList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []DockerCredential `json:"items"`
}

func NewDockerCredential(namespace, name string, obj DockerCredential) *DockerCredential {
	obj.APIVersion, obj.Kind = SchemeGroupVersion.WithKind("DockerCredential").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	return &obj
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NamespacedBasicAuthList is a list of NamespacedBasicAuth resources
type NamespacedBasicAuthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NamespacedBasicAuth `json:"items"`
}

func NewNamespacedBasicAuth(namespace, name string, obj NamespacedBasicAuth) *NamespacedBasicAuth {
	obj.APIVersion, obj.Kind = SchemeGroupVersion.WithKind("NamespacedBasicAuth").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	return &obj
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NamespacedCertificateList is a list of NamespacedCertificate resources
type NamespacedCertificateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NamespacedCertificate `json:"items"`
}

func NewNamespacedCertificate(namespace, name string, obj NamespacedCertificate) *NamespacedCertificate {
	obj.APIVersion, obj.Kind = SchemeGroupVersion.WithKind("NamespacedCertificate").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	return &obj
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NamespacedDockerCredentialList is a list of NamespacedDockerCredential resources
type NamespacedDockerCredentialList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NamespacedDockerCredential `json:"items"`
}

func NewNamespacedDockerCredential(namespace, name string, obj NamespacedDockerCredential) *NamespacedDockerCredential {
	obj.APIVersion, obj.Kind = SchemeGroupVersion.WithKind("NamespacedDockerCredential").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	return &obj
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NamespacedSSHAuthList is a list of NamespacedSSHAuth resources
type NamespacedSSHAuthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NamespacedSSHAuth `json:"items"`
}

func NewNamespacedSSHAuth(namespace, name string, obj NamespacedSSHAuth) *NamespacedSSHAuth {
	obj.APIVersion, obj.Kind = SchemeGroupVersion.WithKind("NamespacedSSHAuth").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	return &obj
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NamespacedServiceAccountTokenList is a list of NamespacedServiceAccountToken resources
type NamespacedServiceAccountTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NamespacedServiceAccountToken `json:"items"`
}

func NewNamespacedServiceAccountToken(namespace, name string, obj NamespacedServiceAccountToken) *NamespacedServiceAccountToken {
	obj.APIVersion, obj.Kind = SchemeGroupVersion.WithKind("NamespacedServiceAccountToken").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	return &obj
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SSHAuthList is a list of SSHAuth resources
type SSHAuthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []SSHAuth `json:"items"`
}

func NewSSHAuth(namespace, name string, obj SSHAuth) *SSHAuth {
	obj.APIVersion, obj.Kind = SchemeGroupVersion.WithKind("SSHAuth").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	return &obj
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceAccountTokenList is a list of ServiceAccountToken resources
type ServiceAccountTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ServiceAccountToken `json:"items"`
}

func NewServiceAccountToken(namespace, name string, obj ServiceAccountToken) *ServiceAccountToken {
	obj.APIVersion, obj.Kind = SchemeGroupVersion.WithKind("ServiceAccountToken").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	return &obj
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkloadList is a list of Workload resources
type WorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Workload `json:"items"`
}

func NewWorkload(namespace, name string, obj Workload) *Workload {
	obj.APIVersion, obj.Kind = SchemeGroupVersion.WithKind("Workload").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	return &obj
}
