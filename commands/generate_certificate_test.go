package commands_test

import (
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/jhanda"
	"github.com/pivotal-cf/om/commands"
	"github.com/pivotal-cf/om/commands/fakes"
)

var _ = Describe("GenerateCertificate", func() {
	var (
		fakeService *fakes.GenerateCertificateService
		fakeLogger  *fakes.Logger
		command     commands.GenerateCertificate
	)

	BeforeEach(func() {
		fakeService = &fakes.GenerateCertificateService{}
		fakeLogger = &fakes.Logger{}
		command = commands.NewGenerateCertificate(fakeService, fakeLogger)
	})

	Describe("Execute", func() {
		It("makes a request to the Opsman to generate a certificate from the given domains", func() {
			err := command.Execute([]string{
				"--domains", "*.apps.example.com, *.sys.example.com",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeService.GenerateCertificateCallCount()).To(Equal(1))
		})

		It("prints a json output for the generated certificate", func() {
			fakeService.GenerateCertificateReturns(`some-json-response`, nil)

			err := command.Execute([]string{
				"--domains", "*.apps.example.com, *.sys.example.com",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeLogger.PrintfCallCount()).To(Equal(1))
			format, content := fakeLogger.PrintfArgsForCall(0)
			Expect(fmt.Sprintf(format, content...)).To(Equal(`some-json-response`))
		})

		Context("failure cases", func() {
			Context("when the domains flag is missing", func() {
				It("returns an error", func() {
					err := command.Execute([]string{})
					Expect(err).To(MatchError("could not parse generate-certificate flags: the required flag `-d, --domains' was not specified"))
				})
			})

			It("returns an error when the service fails to generate a certificate", func() {
				fakeService.GenerateCertificateReturns(`some-json-response`, errors.New("failed to generate certificate"))

				err := command.Execute([]string{
					"--domains", "*.apps.example.com, *.sys.example.com",
				})
				Expect(err).To(MatchError("failed to generate certificate"))
			})
		})
	})

	Describe("Usage", func() {
		It("returns usage info", func() {
			usage := command.Usage()
			Expect(usage).To(Equal(jhanda.Usage{
				Description:      "This authenticated command generates a new RSA public/private certificate signed by Ops Manager’s root CA certificate",
				ShortDescription: "generates a new certificate signed by Ops Manager's root CA",
				Flags:            command.Options,
			}))
		})
	})
})
