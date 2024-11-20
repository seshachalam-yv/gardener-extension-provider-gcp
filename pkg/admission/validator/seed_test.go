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
	mockclient "github.com/gardener/gardener/third_party/mock/controller-runtime/client"
	mockmanager "github.com/gardener/gardener/third_party/mock/controller-runtime/manager"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/gardener-extension-provider-gcp/pkg/admission/validator"
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
		mgr = mockmanager.NewMockManager(ctrl)
		mgr.EXPECT().GetScheme().Return(scheme).Times(1)
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
							Raw: []byte(`{"immutability":{"retentionType":"bucket","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"immutability":{"retentionType":"bucket","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			nil,
		),
		Entry("should not allow update when retention period is reduced",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"immutability":{"retentionType":"bucket","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"immutability":{"retentionType":"bucket","retentionPeriod":"48h"}}`),
						},
					},
				},
			},
			errors.New("reducing the retention period from 96h0m0s to 48h0m0s is not allowed"),
		),
		Entry("should not allow update when retention type is changed",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"immutability":{"retentionType":"bucket","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"immutability":{"retentionType":"bar","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			errors.New("modifying the retention type from 'bucket' to 'bar' is not allowed"),
		),
		Entry("should not allow disabling immutable settings",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"immutability":{"retentionType":"bucket","retentionPeriod":"96h"}}`),
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
							Raw: []byte(`{"immutability":{"retentionType":"bucket","retentionPeriod":"96h"}}`),
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
							Raw: []byte(`{"immutability":{"retentionType":"invalid","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			errors.New("invalid retentionType 'invalid'; must be 'bucket'"),
		),
		Entry("should not allow creation with invalid retention period format",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"immutability":{"retentionType":"bucket","retentionPeriod":"invalid"}}`),
						},
					},
				},
			},
			errors.New("error parsing immutable settings: error parsing immutable settings: invalid duration: time: invalid duration \"invalid\""),
		),
		Entry("should not allow creation with negative retention period",
			&core.Seed{
				Spec: core.SeedSpec{
					Backup: &core.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"immutability":{"retentionType":"bucket","retentionPeriod":"-96h"}}`),
						},
					},
				},
			},
			errors.New("retentionPeriod must be greater than zero"),
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
	)

})
