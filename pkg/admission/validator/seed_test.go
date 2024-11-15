package validator_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/gardener/gardener-extension-provider-gcp/pkg/admission/validator"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	mockclient "github.com/gardener/gardener/third_party/mock/controller-runtime/client"
	mockmanager "github.com/gardener/gardener/third_party/mock/controller-runtime/manager"
	"go.uber.org/mock/gomock"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
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
		Expect(gardencorev1beta1.AddToScheme(scheme)).To(Succeed())
		mgr = mockmanager.NewMockManager(ctrl)
		mgr.EXPECT().GetScheme().Return(scheme).Times(1)
		mgr.EXPECT().GetClient().Return(c)
		seedValidator = validator.NewSeedValidator(mgr)
	})

	DescribeTable("ValidateUpdate",
		func(oldSeed, newSeed *v1beta1.Seed, expectedError error) {
			err := seedValidator.Validate(context.Background(), newSeed, oldSeed)
			if expectedError != nil {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(expectedError.Error()))
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("should allow update when immutable settings are unchanged",
			&v1beta1.Seed{
				Spec: v1beta1.SeedSpec{
					Backup: &v1beta1.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"immutableSettings":{"retentionType":"bucket","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			&v1beta1.Seed{
				Spec: v1beta1.SeedSpec{
					Backup: &v1beta1.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"immutableSettings":{"retentionType":"bucket","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			nil,
		),
		Entry("should not allow update when retention period is reduced",
			&v1beta1.Seed{
				Spec: v1beta1.SeedSpec{
					Backup: &v1beta1.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"immutableSettings":{"retentionType":"bucket","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			&v1beta1.Seed{
				Spec: v1beta1.SeedSpec{
					Backup: &v1beta1.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"immutableSettings":{"retentionType":"bucket","retentionPeriod":"48h"}}`),
						},
					},
				},
			},
			errors.New("reducing the retention period is not allowed"),
		),
		Entry("should not allow update when retention type is changed",
			&v1beta1.Seed{
				Spec: v1beta1.SeedSpec{
					Backup: &v1beta1.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"immutableSettings":{"retentionType":"bucket","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			&v1beta1.Seed{
				Spec: v1beta1.SeedSpec{
					Backup: &v1beta1.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"immutableSettings":{"retentionType":"bar","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			errors.New("modifying the retention type is not allowed"),
		),
		Entry("should not allow disabling immutable settings",
			&v1beta1.Seed{
				Spec: v1beta1.SeedSpec{
					Backup: &v1beta1.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"immutableSettings":{"retentionType":"bucket","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			&v1beta1.Seed{
				Spec: v1beta1.SeedSpec{
					Backup: &v1beta1.SeedBackup{
						ProviderConfig: nil,
					},
				},
			},
			errors.New("disabling immutable settings is not allowed"),
		),
	)

	var _ = DescribeTable("ValidateCreate",
		func(newSeed *v1beta1.Seed, expectedError error) {
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
			&v1beta1.Seed{
				Spec: v1beta1.SeedSpec{
					Backup: &v1beta1.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"immutableSettings":{"retentionType":"bucket","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			nil,
		),
		Entry("should not allow creation with invalid retention type",
			&v1beta1.Seed{
				Spec: v1beta1.SeedSpec{
					Backup: &v1beta1.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"immutableSettings":{"retentionType":"invalid","retentionPeriod":"96h"}}`),
						},
					},
				},
			},
			errors.New("invalid retentionType 'invalid'; must be 'bucket'"),
		),
		Entry("should not allow creation with invalid retention period format",
			&v1beta1.Seed{
				Spec: v1beta1.SeedSpec{
					Backup: &v1beta1.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"immutableSettings":{"retentionType":"bucket","retentionPeriod":"invalid"}}`),
						},
					},
				},
			},
			errors.New("invalid retentionPeriod format: time: invalid duration \"invalid\""),
		),
		Entry("should not allow creation with negative retention period",
			&v1beta1.Seed{
				Spec: v1beta1.SeedSpec{
					Backup: &v1beta1.SeedBackup{
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte(`{"immutableSettings":{"retentionType":"bucket","retentionPeriod":"-96h"}}`),
						},
					},
				},
			},
			errors.New("retentionPeriod must be greater than zero"),
		),
		Entry("should allow creation without immutable settings",
			&v1beta1.Seed{
				Spec: v1beta1.SeedSpec{
					Backup: &v1beta1.SeedBackup{
						ProviderConfig: nil,
					},
				},
			},
			nil,
		),
	)

})
