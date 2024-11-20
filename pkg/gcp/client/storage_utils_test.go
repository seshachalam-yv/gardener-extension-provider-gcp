package client

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("ParseImmutableSettings", func() {
	Context("when providerConfig is nil", func() {
		It("should return nil without error", func() {
			result, err := ParseImmutableSettings(nil)
			Expect(err).To(BeNil())
			Expect(result).To(BeNil())
		})
	})

	Context("when providerConfig is empty", func() {
		It("should return nil without error", func() {
			providerConfig := &runtime.RawExtension{Raw: []byte{}}
			result, err := ParseImmutableSettings(providerConfig)
			Expect(err).To(BeNil())
			Expect(result).To(BeNil())
		})
	})

	Context("when providerConfig contains valid immutability settings", func() {
		It("should parse the settings correctly", func() {

			providerConfig := &runtime.RawExtension{Raw: []byte(`{"immutability":{"retentionType":"bucket","retentionPeriod":"96h"}}`)}
			result, err := ParseImmutableSettings(providerConfig)
			Expect(err).To(BeNil())
			Expect(result).ToNot(BeNil())
			Expect(result.RetentionType).To(Equal(retentionTypeBucket))
			Expect(result.RetentionPeriod).To(Equal(Duration(96 * time.Hour)))
		})
	})

	Context("when providerConfig contains invalid JSON", func() {
		It("should return an error", func() {
			providerConfig := &runtime.RawExtension{Raw: []byte("{invalid-json}")}
			result, err := ParseImmutableSettings(providerConfig)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Context("when providerConfig does not contain immutability settings", func() {
		It("should return nil without error", func() {
			config := struct {
				OtherField string `json:"otherField"`
			}{
				OtherField: "value",
			}
			raw, err := json.Marshal(config)
			Expect(err).To(BeNil())

			providerConfig := &runtime.RawExtension{Raw: raw}
			result, err := ParseImmutableSettings(providerConfig)
			Expect(err).To(BeNil())
			Expect(result).To(BeNil())
		})
	})
})
