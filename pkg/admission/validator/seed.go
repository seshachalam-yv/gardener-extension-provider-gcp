// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"context"
	"errors"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	gcpclient "github.com/gardener/gardener-extension-provider-gcp/pkg/gcp/client"
)

// NewSeedValidator returns a new Validator for Seed resources,
// ensuring backup configuration immutability according to policy.
func NewSeedValidator(mgr manager.Manager) webhook.Validator {
	return &seedValidator{
		client:  mgr.GetClient(),
		decoder: serializer.NewCodecFactory(mgr.GetScheme()).UniversalDecoder(),
	}
}

// seedValidator validates create and update operations on Seed resources,
// enforcing immutability of backup configurations.
type seedValidator struct {
	client  client.Client
	decoder runtime.Decoder
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

	return s.validateCreate(ctx, newSeed)
}

// validateCreate validates the Seed object upon creation.
// It checks if immutable settings are provided and validates them to ensure they meet the required criteria.
func (s *seedValidator) validateCreate(_ context.Context, newSeed *core.Seed) error {
	newImSettings, err := gcpclient.ParseImmutableSettings(newSeed.Spec.Backup.ProviderConfig)
	if err != nil {
		return fmt.Errorf("error parsing immutable settings: %w", err)
	}

	// If immutable settings are provided, validate them
	if newImSettings != nil {
		if err := gcpclient.ValidateImmutableSettings(newImSettings); err != nil {
			return err
		}
	}

	return nil
}

// validateUpdate validates the Seed object during an update operation.
// It ensures that immutability policies are enforced by preventing:
// - Disabling immutable settings once they are enabled.
// - Reducing the retention period.
// - Modifying the retention type.
func (s *seedValidator) validateUpdate(_ context.Context, oldSeed, newSeed *core.Seed) error {
	oldImSettings, err := gcpclient.ParseImmutableSettings(oldSeed.Spec.Backup.ProviderConfig)
	if err != nil {
		return fmt.Errorf("error parsing old immutable settings: %w", err)
	}
	newImSettings, err := gcpclient.ParseImmutableSettings(newSeed.Spec.Backup.ProviderConfig)
	if err != nil {
		return fmt.Errorf("error parsing new immutable settings: %w", err)
	}

	// Ensure that immutable settings are not disabled
	if newImSettings == nil && oldImSettings != nil {
		return errors.New("disabling immutable settings is not allowed")
	}

	// Ensure the retention period is not reduced
	if newImSettings != nil && oldImSettings != nil {
		if newImSettings.RetentionPeriod < oldImSettings.RetentionPeriod {
			return fmt.Errorf("reducing the retention period from %s to %s is not allowed", oldImSettings.RetentionPeriod, newImSettings.RetentionPeriod)
		}

		// Ensure the retention type is not changed
		if newImSettings.RetentionType != oldImSettings.RetentionType {
			return fmt.Errorf("modifying the retention type from '%s' to '%s' is not allowed", oldImSettings.RetentionType, newImSettings.RetentionType)
		}
	}

	// If immutable settings are newly added, validate them
	if oldImSettings == nil && newImSettings != nil {
		if err := gcpclient.ValidateImmutableSettings(newImSettings); err != nil {
			return err
		}
	}

	return nil
}
