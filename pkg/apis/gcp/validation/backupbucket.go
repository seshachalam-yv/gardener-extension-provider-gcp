// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	apisgcp "github.com/gardener/gardener-extension-provider-gcp/pkg/apis/gcp"
)

// ValidateBackupBucketConfig validates a BackupBucketConfig object.
func ValidateBackupBucketConfig(config *apisgcp.BackupBucketConfig, fldPath *field.Path) field.ErrorList {
	if config == nil {
		return nil
	}
	allErrs := field.ErrorList{}

	if config.Immutability.RetentionType != "bucket" {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("immutability").Child("retentionType"), config.Immutability.RetentionType, "retentionType must be 'bucket'"))
	}

	if config.Immutability.RetentionPeriod.Duration <= 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("immutability").Child("retentionPeriod"), config.Immutability.RetentionPeriod.Duration.String(), "retentionPeriod must be a positive duration like '1h', '30m', etc"))
	}

	return allErrs
}
