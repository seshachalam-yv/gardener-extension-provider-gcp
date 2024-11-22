// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BackupBucketConfig represents the configuration for a backup bucket.
type BackupBucketConfig struct {
	metav1.TypeMeta `json:",inline"`

	// Immutability defines the immutability config for the backup bucket.
	Immutability ImmutableConfig `json:"immutability"`
}

// ImmutableConfig represents the immutability configuration for a backup bucket.
type ImmutableConfig struct {
	// RetentionType specifies the type of retention for the backup bucket.
	// +kubebuilder:validation:Enum=bucket
	// +kubebuilder:validation:Immutable
	RetentionType string `json:"retentionType"`

	// RetentionPeriod specifies the retention period for the backup bucket.
	// +kubebuilder:validation:Minimum=1s
	// +kubebuilder:validation:XValidation:rule="self > duration('0s')",message="RetentionPeriod must be a positive duration like '1h', '30m', etc."
	RetentionPeriod metav1.Duration `json:"retentionPeriod"`
}
