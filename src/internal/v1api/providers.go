package v1api

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/opentofu/registry-stable/internal/files"
	"github.com/opentofu/registry-stable/internal/github"
	"github.com/opentofu/registry-stable/internal/provider"
)

type ProviderGenerator struct {
	provider.Provider
	provider.MetadataFile
	Destination string
	log         *slog.Logger
}

func NewProviderGenerator(p provider.Provider, destination string) (ProviderGenerator, error) {
	metadata, err := p.ReadMetadata()
	if err != nil {
		return ProviderGenerator{}, err
	}
	return ProviderGenerator{
		Provider:     p,
		MetadataFile: metadata,
		Destination:  destination,
		log:          p.Logger,
	}, err
}

func (p ProviderGenerator) VersionListingPath() string {
	return filepath.Join(p.Destination, "v1", "providers", p.Provider.Namespace, p.Provider.ProviderName, "versions")
}

func (p ProviderGenerator) VersionDownloadPath(ver provider.Version, details ProviderVersionDetails) string {
	return filepath.Join(p.Destination, "v1", "providers", p.Provider.Namespace, p.Provider.ProviderName, ver.Version, "download", details.OS, details.Arch)
}

func (p ProviderGenerator) VersionListing() ProviderVersionListingResponse {
	versions := make([]ProviderVersionResponseItem, len(p.MetadataFile.Versions))

	for versionIdx, ver := range p.MetadataFile.Versions {
		verResp := ProviderVersionResponseItem{
			Version:   ver.Version,
			Protocols: ver.Protocols,
			Platforms: make([]github.Platform, len(ver.Targets)),
		}

		for targetIdx, target := range ver.Targets {
			verResp.Platforms[targetIdx] = github.Platform{
				OS:   target.OS,
				Arch: target.Arch,
			}
		}
		versions[versionIdx] = verResp
	}

	return ProviderVersionListingResponse{versions}
}

func (p ProviderGenerator) VersionDetails() map[string]ProviderVersionDetails {
	versionDetails := make(map[string]ProviderVersionDetails)

	for _, ver := range p.MetadataFile.Versions {
		for _, target := range ver.Targets {
			details := ProviderVersionDetails{
				Protocols:           ver.Protocols,
				OS:                  target.OS,
				Arch:                target.Arch,
				Filename:            target.Filename,
				DownloadURL:         target.DownloadURL,
				SHASumsURL:          ver.SHASumsURL,
				SHASumsSignatureURL: ver.SHASumsSignatureURL,
				SHASum:              target.SHASum,
				SigningKeys:         SigningKeys{}, // TODO: Add gpg keys
			}
			versionDetails[p.VersionDownloadPath(ver, details)] = details
		}
	}
	return versionDetails
}

// Generate generates the response for the provider version listing API endpoints.
func (p ProviderGenerator) Generate() error {
	p.log.Info("Generating")

	for location, details := range p.VersionDetails() {
		err := files.SafeWriteObjectToJSONFile(location, details)
		if err != nil {
			return fmt.Errorf("failed to write metadata version download file: %w", err)
		}
	}

	err := files.SafeWriteObjectToJSONFile(p.VersionListingPath(), p.VersionListing())
	if err != nil {
		return err
	}

	p.log.Info("Generated")

	return nil
}
