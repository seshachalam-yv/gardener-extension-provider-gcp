// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"context"
	"fmt"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-provider-gcp/pkg/admission"
	"github.com/gardener/gardener-extension-provider-gcp/pkg/apis/gcp"
)

// NewSeedValidator returns a new Validator for Seed resources,
// ensuring backup configuration immutability according to policy.
func NewSeedValidator(mgr manager.Manager) extensionswebhook.Validator {
	return &seedValidator{
		client:         mgr.GetClient(),
		decoder:        serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),
		lenientDecoder: serializer.NewCodecFactory(mgr.GetScheme()).UniversalDecoder(),
	}
}

// seedValidator validates create and update operations on Seed resources,
// enforcing immutability of backup configurations.
type seedValidator struct {
	client         client.Client
	decoder        runtime.Decoder
	lenientDecoder runtime.Decoder
}

// Validate validates the Seed resource during create or update operations.
// It enforces immutability policies on backup configurations to prevent
// disabling immutable settings, reducing retention periods, or changing retention types.
func (s *seedValidator) Validate(ctx context.Context, newObj, oldObj client.Object) error {
	newSeed, ok := newObj.(*core.Seed)
	if !ok {
		return fmt.Errorf("wrong object type %T for new object", newObj)
	}

	if oldObj != nil {
		oldSeed, ok := oldObj.(*core.Seed)
		if !ok {
			return fmt.Errorf("wrong object type %T for old object", oldObj)
		}
		return s.validateUpdate(ctx, oldSeed, newSeed)
	}

	return s.validateCreate(newSeed)
}

// validateCreate validates the Seed object upon creation.
// It checks if immutable settings are provided and validates them to ensure they meet the required criteria.
func (s *seedValidator) validateCreate(newSeed *core.Seed) error {
	_, err := admission.DecodeBackupBucketConfig(s.decoder, newSeed.Spec.Backup.ProviderConfig)
	if err != nil {
		return fmt.Errorf("error decoding BackupBucketConfig: %w", err)
	}

	return nil
}

// validateUpdate validates the Seed object during an update operation.
// It ensures that immutability policies are enforced by preventing:
// - Disabling immutable settings once they are enabled.
// - Reducing the retention period.
// - Modifying the retention type.
func (s *seedValidator) validateUpdate(_ context.Context, oldSeed, newSeed *core.Seed) error {
	oldbackupBucketConfig, err := admission.DecodeBackupBucketConfig(s.lenientDecoder, oldSeed.Spec.Backup.ProviderConfig)
	if err != nil {
		return fmt.Errorf("error decoding old BackupBucketConfig: %w", err)
	}

	newBackupBucketConfig, err := admission.DecodeBackupBucketConfig(s.decoder, newSeed.Spec.Backup.ProviderConfig)
	if err != nil {
		return fmt.Errorf("error decoding new BackupBucketConfig: %w", err)
	}

	// Ensure that immutable settings are not disabled
	if (newBackupBucketConfig == nil || newBackupBucketConfig.Immutability == (gcp.ImmutableConfig{})) && oldbackupBucketConfig.Immutability != (gcp.ImmutableConfig{}) {
		return fmt.Errorf("disabling immutable settings is not allowed")
	}

	// Ensure the retention period is not reduced
	if newBackupBucketConfig.Immutability != (gcp.ImmutableConfig{}) && oldbackupBucketConfig.Immutability != (gcp.ImmutableConfig{}) {
		if newBackupBucketConfig.Immutability.RetentionPeriod.Duration < oldbackupBucketConfig.Immutability.RetentionPeriod.Duration {
			return fmt.Errorf("reducing the retention period from %v to %v is not allowed. Please ensure the new retention period is greater than or equal to the old retention period", oldbackupBucketConfig.Immutability.RetentionPeriod.Duration, newBackupBucketConfig.Immutability.RetentionPeriod.Duration)
		}

		// Ensure the retention type is not changed
		if newBackupBucketConfig.Immutability.RetentionType != oldbackupBucketConfig.Immutability.RetentionType {
			return fmt.Errorf("modifying the retention type from '%s' to '%s' is not allowed", oldbackupBucketConfig.Immutability.RetentionType, newBackupBucketConfig.Immutability.RetentionType)
		}
	}

	return nil
}
