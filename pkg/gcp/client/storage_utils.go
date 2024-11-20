// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
)

// Duration wraps time.Duration to allow custom JSON unmarshaling.
type Duration time.Duration

// UnmarshalJSON implements the json.Unmarshaler interface for Duration.
func (d *Duration) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	duration, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration: %v", err)
	}
	*d = Duration(duration)
	return nil
}

// String implements the fmt.Stringer interface for Duration.
func (d Duration) String() string {
	return time.Duration(d).String()
}

// ToDuration converts Duration to time.Duration.
func (d Duration) ToDuration() time.Duration {
	return time.Duration(d)
}

// Immutability represents the settings for immutable storage.
type Immutability struct {
	RetentionType   string   `json:"retentionType"`
	RetentionPeriod Duration `json:"retentionPeriod"`
}

// ParseImmutableSettings decodes providerConfig into Immutability.
func ParseImmutableSettings(providerConfig *runtime.RawExtension) (*Immutability, error) {
	if providerConfig == nil || len(providerConfig.Raw) == 0 {
		return nil, nil // No immutable settings provided
	}

	var config struct {
		Immutability *Immutability `json:"immutability"`
	}

	if err := json.Unmarshal(providerConfig.Raw, &config); err != nil {
		return nil, fmt.Errorf("error parsing immutable settings: %v", err)
	}

	if config.Immutability == nil {
		return nil, nil // No immutable settings provided
	}

	return config.Immutability, nil
}

const RetentionTypeBucket = "bucket"

// ValidateImmutableSettings checks the validity of Immutability settings.
func ValidateImmutableSettings(imSettings *Immutability) error {
	if imSettings == nil {
		return errors.New("immutable settings cannot be nil")
	}

	// Validate RetentionType
	if imSettings.RetentionType != RetentionTypeBucket {
		return fmt.Errorf("invalid retentionType '%s'; must be '%s'", imSettings.RetentionType, RetentionTypeBucket)
	}

	// Validate RetentionPeriod
	if imSettings.RetentionPeriod.ToDuration() <= 0 {
		return errors.New("retentionPeriod must be greater than zero")
	}

	return nil
}
