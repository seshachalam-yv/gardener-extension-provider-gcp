package validator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener/extensions/pkg/webhook"
)

// seedValidator validates create and update operations on Seed resources.
type seedValidator struct {
	client  client.Client
	decoder runtime.Decoder
}

// ImmutableSettings represents the settings for immutable storage.
type ImmutableSettings struct {
	RetentionType   string
	RetentionPeriod time.Duration
}

// ImmutableSettingsJSON is used for JSON unmarshalling of immutable settings.
type ImmutableSettingsJSON struct {
	RetentionType   string `json:"retentionType"`
	RetentionPeriod string `json:"retentionPeriod"`
}

// NewSeedValidator returns a new instance of a seed validator.
func NewSeedValidator(mgr manager.Manager) webhook.Validator {
	return &seedValidator{
		client:  mgr.GetClient(),
		decoder: serializer.NewCodecFactory(mgr.GetScheme()).UniversalDecoder(),
	}
}

// Validate validates the given Seed objects.
func (s *seedValidator) Validate(ctx context.Context, newObj, oldObj client.Object) error {
	newSeed, ok := newObj.(*v1beta1.Seed)
	if !ok {
		return fmt.Errorf("wrong object type %T for new object", newObj)
	}

	if oldObj != nil {
		oldSeed, ok := oldObj.(*v1beta1.Seed)
		if !ok {
			return fmt.Errorf("wrong object type %T for old object", oldObj)
		}
		return s.validateUpdate(ctx, oldSeed, newSeed)
	}

	return s.validateCreate(ctx, newSeed)
}

// validateCreate validates the Seed object upon creation.
func (s *seedValidator) validateCreate(_ context.Context, newSeed *v1beta1.Seed) error {
	newImSettings, err := ParseImmutableSettings(newSeed.Spec.Backup.ProviderConfig)
	if err != nil {
		return fmt.Errorf("error parsing immutable settings: %v", err)
	}

	// If immutable settings are provided, validate them
	if newImSettings != nil {
		if err := validateImmutableSettings(newImSettings); err != nil {
			return err
		}
	}

	// Additional validation can be added here if needed
	return nil
}

// validateUpdate checks that certain fields in a Seed's backup configuration are immutable when specified.
func (s *seedValidator) validateUpdate(_ context.Context, oldSeed, newSeed *v1beta1.Seed) error {
	oldImSettings, err := ParseImmutableSettings(oldSeed.Spec.Backup.ProviderConfig)
	if err != nil {
		return fmt.Errorf("error parsing old immutable settings: %v", err)
	}
	newImSettings, err := ParseImmutableSettings(newSeed.Spec.Backup.ProviderConfig)
	if err != nil {
		return fmt.Errorf("error parsing new immutable settings: %v", err)
	}

	// Ensure that immutable settings are not disabled
	if newImSettings == nil && oldImSettings != nil {
		return errors.New("disabling immutable settings is not allowed")
	}

	// Ensure the retention period is not reduced
	if newImSettings != nil && oldImSettings != nil {
		if newImSettings.RetentionPeriod < oldImSettings.RetentionPeriod {
			return errors.New("reducing the retention period is not allowed")
		}

		// Ensure other aspects of immutable settings have not been altered
		if newImSettings.RetentionType != oldImSettings.RetentionType {
			return errors.New("modifying the retention type is not allowed")
		}
	}

	// If immutable settings are newly added, validate them
	if oldImSettings == nil && newImSettings != nil {
		if err := validateImmutableSettings(newImSettings); err != nil {
			return err
		}
	}

	return nil
}

// validateImmutableSettings performs additional checks on ImmutableSettings.
func validateImmutableSettings(imSettings *ImmutableSettings) error {
	// Validate RetentionType
	if imSettings.RetentionType != "bucket" {
		return fmt.Errorf("invalid retentionType '%s'; must be 'bucket'", imSettings.RetentionType)
	}

	// Validate RetentionPeriod
	if imSettings.RetentionPeriod <= 0 {
		return errors.New("retentionPeriod must be greater than zero")
	}

	return nil
}

// ParseImmutableSettings decodes providerConfig into ImmutableSettings.
func ParseImmutableSettings(providerConfig *runtime.RawExtension) (*ImmutableSettings, error) {
	if providerConfig == nil || len(providerConfig.Raw) == 0 {
		return nil, nil // No immutable settings provided
	}

	var config struct {
		ImmutableSettings *ImmutableSettingsJSON `json:"immutableSettings"`
	}

	err := json.Unmarshal(providerConfig.Raw, &config)
	if err != nil {
		return nil, fmt.Errorf("error while parsing immutable settings: %v", err)
	}

	if config.ImmutableSettings == nil {
		return nil, nil // No immutable settings provided
	}

	duration, err := time.ParseDuration(config.ImmutableSettings.RetentionPeriod)
	if err != nil {
		return nil, fmt.Errorf("invalid retentionPeriod format: %v", err)
	}

	return &ImmutableSettings{
		RetentionType:   config.ImmutableSettings.RetentionType,
		RetentionPeriod: duration,
	}, nil
}
