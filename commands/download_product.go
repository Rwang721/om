package commands

import (
	"fmt"
	"github.com/pivotal-cf/go-pivnet"
	log "github.com/pivotal-cf/go-pivnet/logger"
	"github.com/pivotal-cf/jhanda"
	"github.com/pivotal-cf/pivnet-cli/filter"
	"io"
	"os"
	"path"
)

//go:generate counterfeiter -o ./fakes/pivnet_client_factory.go --fake-name PivnetClientFactory . PivnetClientFactory
type PivnetClientFactory interface {
	NewClient(config pivnet.ClientConfig, logger log.Logger) PivnetDownloader
}

//go:generate counterfeiter -o ./fakes/pivnet_downloader_service.go --fake-name PivnetDownloader . PivnetDownloader
type PivnetDownloader interface {
	ReleaseForVersion(productSlug string, releaseVersion string) (pivnet.Release, error)
	ProductFilesForRelease(productSlug string, releaseID int) ([]pivnet.ProductFile, error)
	DownloadProductFile(location *os.File, productSlug string, releaseID int, productFileID int, progressWriter io.Writer) error
}

type DownloadProduct struct {
	logger         log.Logger
	progressWriter io.Writer
	pivnetFactory  PivnetClientFactory
	filter         *filter.Filter
	Options        struct {
		ConfigFile     string `long:"config"               short:"c"   description:"path to yml file for configuration (keys must match the following command line flags)"`
		Token          string `long:"pivnet-api-token"                  required:"true"`
		FileGlob       string `long:"pivnet-file-glob"     short:"f"   description:"Glob to match files within Pivotal Network product to be downloaded." required:"true"`
		ProductSlug    string `long:"pivnet-product-slug"  short:"p"   description:"path to product" required:"true"`
		ProductVersion string `long:"product-version"                  description:"version of the provided product file to be used for validation" required:"true"`
		OutputDir      string `long:"output-directory"     short:"o"   description:"Directory path to which the file will be outputted. File name will be preserved from Pivotal Network."`
	}
}

func NewDownloadProduct(logger log.Logger, progressWriter io.Writer, factory PivnetClientFactory) DownloadProduct {
	return DownloadProduct{
		logger:         logger,
		progressWriter: progressWriter,
		pivnetFactory:  factory,
		filter:         filter.NewFilter(logger),
	}
}

func (c DownloadProduct) Usage() jhanda.Usage {
	return jhanda.Usage{
		Description:      "This command attempts to download a single product file from Pivotal Network. The API token used must be associated with a user account that has already accepted the EULA for the specified product",
		ShortDescription: "downloads a specified product file from Pivotal Network",
		Flags:            c.Options,
	}
}

func (c DownloadProduct) Execute(args []string) error {
	err := loadConfigFile(args, &c.Options)
	if err != nil {
		return fmt.Errorf("could not parse download-product flags: %s", err)
	}

	config := pivnet.ClientConfig{
		Host:              pivnet.DefaultHost,
		Token:             c.Options.Token,
		UserAgent:         "om",
		SkipSSLValidation: false,
	}

	client := c.pivnetFactory.NewClient(config, c.logger)

	release, err := client.ReleaseForVersion(c.Options.ProductSlug, c.Options.ProductVersion)
	productFiles, err := client.ProductFilesForRelease(c.Options.ProductSlug, release.ID)
	a, err := c.filter.ProductFileKeysByGlobs(productFiles, []string{c.Options.FileGlob})
	if len(a) > 1 {
		var productFileNames []string
		for _, productFile := range productFiles {
			productFileNames = append(productFileNames, productFile.Name)
		}
		return fmt.Errorf("the glob '%s' matches multiple files. Write your glob to match exactly one of the following: %v", c.Options.FileGlob, productFileNames)
	} else if len(a) == 0 {
		return fmt.Errorf("the glob '%s' matchs no file", c.Options.FileGlob)
	}

	productFile := productFiles[0]

	OutputFile, err := os.Create(path.Join(c.Options.OutputDir, productFile.Name))

	err = client.DownloadProductFile(OutputFile, c.Options.ProductSlug, release.ID, productFile.ID, c.progressWriter)
	return nil
}