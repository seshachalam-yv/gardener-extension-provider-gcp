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

// Duration is a custom type that wraps time.Duration to allow custom JSON unmarshaling.
// It enables parsing duration strings (e.g., "96h") from JSON into time.Duration values.
type Duration time.Duration

// UnmarshalJSON implements the json.Unmarshaler interface for the Duration type.
// It allows the Duration type to be unmarshaled from JSON strings representing time durations.
func (d *Duration) UnmarshalJSON(b []byte) error {
	// Convert the JSON byte array to a string and trim quotes
	s := strings.Trim(string(b), "\"")
	// Parse the duration string into a time.Duration
	duration, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration: %v", err)
	}
	// Assign the parsed duration to the receiver
	*d = Duration(duration)
	return nil
}

// String returns the string representation of the Duration.
// It implements the fmt.Stringer interface.
func (d Duration) String() string {
	return time.Duration(d).String()
}

func (d Duration) toDuration() time.Duration {
	return time.Duration(d)
}

// Immutability represents the settings for immutable storage configurations.
// It includes the retention type and the retention period for the backup bucket.
type Immutability struct {
	RetentionType   string   `json:"retentionType"`
	RetentionPeriod Duration `json:"retentionPeriod"`
}

// retentionTypeBucket is a constant representing the "bucket" retention type.
// It specifies that the retention policy applies to the entire bucket.
const retentionTypeBucket = "bucket"

// ParseImmutableSettings parses the providerConfig into an Immutability struct.
// It extracts the immutability settings from the raw JSON configuration.
func ParseImmutableSettings(providerConfig *runtime.RawExtension) (*Immutability, error) {
	// Check if the providerConfig is nil or empty
	if providerConfig == nil || len(providerConfig.Raw) == 0 {
		return nil, nil // No immutable settings provided
	}

	// Define a temporary struct to unmarshal the immutability settings
	var config struct {
		Immutability *Immutability `json:"immutability"`
	}

	// Unmarshal the raw JSON into the config struct
	if err := json.Unmarshal(providerConfig.Raw, &config); err != nil {
		return nil, fmt.Errorf("error parsing immutable settings: %v", err)
	}

	// If no immutability settings are provided, return nil
	if config.Immutability == nil {
		return nil, nil // No immutable settings provided
	}

	// Return the parsed immutability settings
	return config.Immutability, nil
}

// ValidateImmutableSettings validates the Immutability configuration.
// It checks that the retention type and retention period are set correctly.
func ValidateImmutableSettings(imSettings *Immutability) error {
	if imSettings == nil {
		return errors.New("immutable settings cannot be nil")
	}

	// Validate the RetentionType
	if imSettings.RetentionType != retentionTypeBucket {
		return fmt.Errorf("invalid retentionType '%s'; must be '%s'", imSettings.RetentionType, retentionTypeBucket)
	}

	if imSettings.RetentionPeriod.toDuration() <= 0 {
		return errors.New("retentionPeriod must be greater than zero")
	}

	// All validations passed
	return nil
}
