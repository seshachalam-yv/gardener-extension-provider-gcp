// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validator_test

import (
	"context"
	"errors"
	"fmt"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	mockclient "github.com/gardener/gardener/third_party/mock/controller-runtime/client"
	mockmanager "github.com/gardener/gardener/third_party/mock/controller-runtime/manager"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/gardener-extension-provider-gcp/pkg/admission/validator"
	apisgcp "github.com/gardener/gardener-extension-provider-gcp/pkg/apis/gcp"
	apisgcpv1alpha1 "github.com/gardener/gardener-extension-provider-gcp/pkg/apis/gcp/v1alpha1"
)

var _ = Describe("Seed Validator", func() {
	var (
		mgr           *mockmanager.MockManager
		seedValidator extensionswebhook.Validator
		ctrl          *gomock.Controller
		c             *mockclient.MockClient
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		scheme := runtime.NewScheme()
		Expect(core.AddToScheme(scheme)).To(Succeed())
		Expect(apisgcp.AddToScheme(scheme)).To(Succeed())
		Expect(apisgcpv1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(gardencorev1beta1.AddToScheme(scheme)).To(Succeed())
		c = mockclient.NewMockClient(ctrl)

		mgr = mockmanager.NewMockManager(ctrl)
		mgr.EXPECT().GetScheme().Return(scheme).Times(2)
		mgr.EXPECT().GetClient().Return(c)
		seedValidator = validator.NewSeedValidator(mgr)
	})

	DescribeTable("ValidateUpdate",
		func(oldSeed, newSeed *core.Seed, expectedError error) {
			err := seedValidator.Validate(context.Background(), newSeed, oldSeed)
			if expectedError != nil {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(expectedError.Error()))
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("should allow update when immutable settings are unchanged",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"bucket","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"bucket","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			nil,
		),
		Entry("should not allow disabling immutable settings",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"bucket","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: nil,
					},
				},
			},
			errors.New("disabling immutable settings is not allowed"),
		),
		Entry("should return error when old BackupBucketConfig decoding fails",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`invalid`),
						},
					},
				},
			},
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"bucket","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			errors.New("error decoding old BackupBucketConfig: couldn't get version/kind; json parse error: json: cannot unmarshal string into Go value of type struct { APIVersion string \"json:\\\"apiVersion,omitempty\\\"\"; Kind string \"json:\\\"kind,omitempty\\\"\" }"),
		),
		Entry("should return error when new BackupBucketConfig decoding fails",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"bucket","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`invalid`),
						},
					},
				},
			},
			errors.New("error decoding new BackupBucketConfig: couldn't get version/kind; json parse error: json: cannot unmarshal string into Go value of type struct { APIVersion string \"json:\\\"apiVersion,omitempty\\\"\"; Kind string \"json:\\\"kind,omitempty\\\"\" }"),
		),
		Entry("should return error when new BackupBucketConfig validation fails",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"bucket","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"invalid","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			errors.New("validation failed: spec.backup.providerConfig.immutability.retentionType: Invalid value: \"invalid\": retentionType must be 'bucket'"),
		),
		Entry("should not allow unlocking immutable retention policy lock",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"bucket","retentionPeriod":"96h","locked":true}}`),
						},
					},
				},
			},
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"bucket","retentionPeriod":"96h","locked":false}}`),
						},
					},
				},
			},
			errors.New("immutable retention policy lock cannot be unlocked once it is locked. Please ensure the retention policy lock remains locked"),
		),
		Entry("should allow update when retention period is unchanged and lock remains true",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"bucket","retentionPeriod":"96h","locked":true}}`),
						},
					},
				},
			},
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"bucket","retentionPeriod":"96h","locked":true}}`),
						},
					},
				},
			},
			nil,
		),
		Entry("should allow update when retention period is increased and lock remains true",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"bucket","retentionPeriod":"96h","locked":true}}`),
						},
					},
				},
			},
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"bucket","retentionPeriod":"120h","locked":true}}`),
						},
					},
				},
			},
			nil,
		),
		Entry("should allow decreasing the retention period if it's not locked",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"bucket","retentionPeriod":"96h","locked":false}}`),
						},
					},
				},
			},
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"bucket","retentionPeriod":"48h","locked":false}}`),
						},
					},
				},
			},
			nil,
		),
		Entry("should not allow decreasing the retention period if it's locked",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"bucket","retentionPeriod":"96h","locked":true}}`),
						},
					},
				},
			},
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"bucket","retentionPeriod":"48h","locked":true}}`),
						},
					},
				},
			},
			errors.New("reducing the retention period from 96h0m0s to 48h0m0s is prohibited when the immutable retention policy is locked. Ensure the new retention period is not shorter than the existing one"),
		),
	)

	var _ = DescribeTable("ValidateCreate",
		func(newSeed *core.Seed, expectedError error) {
			err := seedValidator.Validate(context.Background(), newSeed, nil)
			if expectedError != nil {
				Expect(err).To(HaveOccurred())
				fmt.Println(err)
				fmt.Println(expectedError)
				Expect(err.Error()).Should(ContainSubstring(expectedError.Error()))
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("should allow creation with valid immutable settings",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"bucket","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			nil,
		),
		Entry("should not allow creation with invalid retention type",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"invalid","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			errors.New("validation failed: spec.backup.providerConfig.immutability.retentionType: Invalid value: \"invalid\": retentionType must be 'bucket'"),
		),
		Entry("should not allow creation with invalid retention period format",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"bucket","retentionPeriod":"invalid"}}`),
						},
					},
				},
			},
			errors.New("error decoding BackupBucketConfig: time: invalid duration \"invalid\""),
		),
		Entry("should not allow creation with negative retention period",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"bucket","retentionPeriod":"-96h"}}`),
						},
					},
				},
			},
			errors.New("validation failed: spec.backup.providerConfig.immutability.retentionPeriod: Invalid value: \"-96h0m0s\": retentionPeriod must be a positive duration greater than 24h"),
		),
		Entry("should allow creation without immutable settings",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: nil,
					},
				},
			},
			nil,
		),
		Entry("should allow creation with locked immutable settings",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","kind":"BackupBucketConfig","immutability":{"retentionType":"bucket","retentionPeriod":"96h","locked":true}}`),
						},
					},
				},
			},
			nil,
		),
	)
})
