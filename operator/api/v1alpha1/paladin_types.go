/*
Copyright 2024.

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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PaladinSpec defines the desired state of Paladin
type PaladinSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Settings from this config will be loaded as YAML and used as the base of the configuration.
	Config *string `json:"config,omitempty"`

	// Database section k8s native functions for setting up the database
	// with auto-generation/auto-edit of the DB related config sections
	Database Database `json:"database,omitempty"`

	// Adds signing modules that load their key materials from a k8s secret
	SecretBackedSigners []SecretBackedSigner `json:"secretBackedSigners,omitempty"`

	// Optionally bind to a local besu node deployed with this operator
	// (vs. configuring a connection to a production blockchain network)
	BesuNode string `json:"besuNode,omitempty"`

	// Optionally tune the service definition.
	// We merge any configuration you add (such as node ports) for the following services:
	// "rpc-http" - 8545 (TCP),
	// "rpc-ws" - 8546 (TCP)
	Service corev1.ServiceSpec `json:"service,omitempty"`
}

const DBMode_EmbeddedSQLite = "embeddedSQLite"
const DBMode_SidecarPostgres = "sidecarPostgres"
const DBMigrationMode_Auto = "auto"

// Database configuration
type Database struct {
	// +kubebuilder:validation:Enum=preConfigured;sidecarPostgres;embeddedSQLite
	// +kubebuilder:default=preConfigured
	Mode string `json:"mode,omitempty"`
	// +kubebuilder:validation:Enum=preConfigured;auto
	// +kubebuilder:default=preConfigured
	MigrationMode string `json:"migrationMode,omitempty"`
	// If set then {{.username}} and {{.password}} variables will be available in your DSN
	PasswordSecret *string                          `json:"passwordSecret,omitempty"`
	PVCTemplate    corev1.PersistentVolumeClaimSpec `json:"pvcTemplate,omitempty"`
}

const SignerType_AutoHDWallet = "autoHDWallet"

type SecretBackedSigner struct {
	Secret string `json:"secret"`
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	Name string `json:"name"` // TODO: Currently only one signer supported in Paladin until key manager in place
	// +kubebuilder:validation:Enum=autoHDWallet;preConfigured
	// +kubebuilder:default=autoHDWallet
	// The operator supports generating the seed and base config for a simple seeded BIP32 HDWallet signer.
	// If more other options are needed, these can be set directly in the YAML config for this signer.
	Type string `json:"type"`
}

// StatusReason is an enumeration of possible failure causes.  Each StatusReason
// must map to a single HTTP status code, but multiple reasons may map
// to the same HTTP status code.
// TODO: move to apiserver
type StatusReason string

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Paladin is the Schema for the paladin API
type Paladin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PaladinSpec `json:"spec,omitempty"`
	Status Status      `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PaladinList contains a list of Paladin
type PaladinList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Paladin `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Paladin{}, &PaladinList{})
}
