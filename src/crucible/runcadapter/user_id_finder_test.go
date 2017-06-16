package runcadapter_test

import (
	"crucible/runcadapter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

var _ = Describe("UserIDFinder", func() {
	var userIDFinder runcadapter.UserIDFinder

	BeforeEach(func() {
		userIDFinder = runcadapter.NewUserIDFinder()
	})

	Context("Lookup", func() {
		It("returns a runc spec User", func() {
			user, err := userIDFinder.Lookup("vcap")
			Expect(err).NotTo(HaveOccurred())
			Expect(user).To(Equal(specs.User{
				UID:      2000,
				GID:      3000,
				Username: "vcap",
			}))
		})

		Context("when the user lookup fails", func() {
			It("returns an error", func() {
				_, err := userIDFinder.Lookup("")
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
