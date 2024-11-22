// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

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
	RetentionType string `json:"retentionType"`

	// RetentionPeriod specifies the retention period for the backup bucket.
	RetentionPeriod metav1.Duration `json:"retentionPeriod"`
}
