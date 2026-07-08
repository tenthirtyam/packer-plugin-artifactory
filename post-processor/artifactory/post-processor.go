// Copyright (c) Ryan Johnson
// SPDX-License-Identifier: MPL-2.0

package artifactory

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

// builderId is the unique identifier for this post-processor.
const builderId = "packer.post-processor.artifactory"

// artifactoryClientInterface defines the interface for Artifactory client
// operations.
type artifactoryClientInterface interface {
	UploadFile(ctx context.Context, filePath, filename string) error
	UploadMetadata(ctx context.Context, metadata artifactMetadata) error
	SetPathProperties(ctx context.Context) error
}

// PostProcessor uploads virtual machine artifacts to Artifactory.
type PostProcessor struct {
	config artifactoryConfig
	client artifactoryClientInterface
}

// ConfigSpec returns the HCL2 configuration specification.
func (p *PostProcessor) ConfigSpec() hcldec.ObjectSpec {
	return p.config.FlatMapstructure().HCL2Spec()
}

// Configure parses and validates the post-processor configuration.
func (p *PostProcessor) Configure(raws ...any) error {
	err := config.Decode(&p.config, &config.DecodeOpts{
		PluginType:  builderId,
		Interpolate: true,
		// EnableEnv allows `{{ env `VAR` }}` in config values (e.g. properties)
		// so CI-provided values like build numbers or commit SHAs can be
		// injected as Artifactory metadata.
		InterpolateContext: &interpolate.Context{EnableEnv: true},
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{},
		},
	}, raws...)
	if err != nil {
		return err
	}

	if err := p.config.Validate(); err != nil {
		return err
	}

	p.client = nil

	return nil
}

// PostProcess uploads artifacts to Artifactory using the extensible handler
// system.
func (p *PostProcessor) PostProcess(ctx context.Context, ui packer.Ui, artifact packer.Artifact) (packer.Artifact, bool, bool, error) {
	ui.Say("Starting Artifactory post-processor...")
	ui.Say(fmt.Sprintf("Artifactory URL: %s", p.config.URL))
	ui.Say(fmt.Sprintf("Repository: %s", p.config.Repository))
	ui.Say(fmt.Sprintf("Builder: %s", artifact.BuilderId()))

	if p.client == nil {
		client, err := newArtifactoryClient(p.config, ui)
		if err != nil {
			return nil, false, false, fmt.Errorf("failed to create Artifactory client: %w", err)
		}
		p.client = client
	}

	processor := newArtifactProcessor(p.config.AdditionalOvfExtensions)

	processedArtifact, err := processor.processArtifact(artifact)
	if err != nil {
		return nil, false, false, fmt.Errorf("failed to process artifact: %w", err)
	}

	ui.Say(fmt.Sprintf("Processing %d files with handler", len(processedArtifact.Files)))

	for _, file := range processedArtifact.Files {
		filename := filepath.Base(file)
		ui.Say(fmt.Sprintf("Uploading file: %s", filename))

		err := p.client.UploadFile(ctx, file, filename)
		if err != nil {
			ui.Error(fmt.Sprintf("Failed to upload file %s: %s", filename, err))
			return nil, false, false, err
		}

		ui.Say(fmt.Sprintf("Successfully uploaded %s to Artifactory", filename))
	}

	metadata := artifactMetadata{
		Name:        p.config.ArtifactName,
		Type:        processedArtifact.Metadata["type"],
		BuilderId:   artifact.BuilderId(),
		Files:       strings.Split(processedArtifact.Metadata["files"], ","),
		Timestamp:   getCurrentTimestamp(),
		Description: fmt.Sprintf("Created by Packer build: %s", artifact.String()),
	}

	err = p.client.UploadMetadata(ctx, metadata)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to upload artifact metadata: %s", err))
		return nil, false, false, err
	}

	ui.Say("Successfully uploaded artifact metadata to Artifactory")

	if err := p.client.SetPathProperties(ctx); err != nil {
		ui.Error(fmt.Sprintf("Failed to set properties on artifact_path: %s", err))
		return nil, false, false, err
	}

	return artifact, false, false, nil
}

// getCurrentTimestamp returns the current Unix timestamp as a string.
func getCurrentTimestamp() string {
	return fmt.Sprintf("%d", time.Now().Unix())
}
