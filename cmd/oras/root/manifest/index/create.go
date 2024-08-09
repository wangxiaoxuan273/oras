/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package index

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/cmd/oras/internal/command"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/descriptor"
)

type createOptions struct {
	option.Common
	option.Target

	sources   []string
	extraRefs []string
}

func createCmd() *cobra.Command {
	var opts createOptions
	cmd := &cobra.Command{
		Use:   "create [flags] <name>[:<tag[,<tag>][...]] [{<tag>|<digest>}...]",
		Short: "Create and push an index from provided manifests",
		Long: `Create and push an index to a repository or an OCI image layout

Example - create an index from source manifests tagged amd64, arm64, darwin in the repository
 localhost:5000/hello, and push the index without tagging it:
  oras manifest index create localhost:5000/hello amd64 arm64 darwin

Example - create an index from source manifests tagged amd64, arm64, darwin in the repository
 localhost:5000/hello, and push the index with tag 'latest':
  oras manifest index create localhost:5000/hello:latest amd64 arm64 darwin

Example - create an index from source manifests using both tags and digests, 
 and push the index with tag 'latest':
  oras manifest index create localhost:5000/hello latest amd64 sha256:xxx darwin

Example - create an index and push it with multiple tags:
  oras manifest index create localhost:5000/tag1, tag2, tag3 amd64 arm64 sha256:xxx
`,
		Args: cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			refs := strings.Split(args[0], ",")
			opts.RawReference = refs[0]
			opts.extraRefs = refs[1:]
			opts.sources = args[1:]
			return option.Parse(cmd, &opts)
			// todo: add EnsureReferenceNotEmpty somewhere
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return createIndex(cmd, opts)
		},
	}

	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.Target)
}

func createIndex(cmd *cobra.Command, opts createOptions) error {
	ctx, logger := command.GetLogger(cmd, &opts.Common)
	target, err := opts.NewTarget(opts.Common, logger)
	if err != nil {
		return err
	}
	// we assume that the sources and the to be created index are all in the same
	// repository, so no copy is needed
	manifests, err := resolveSourceManifests(ctx, target, opts.sources)
	if err != nil {
		return err
	}
	desc, content, err := packIndex(&ocispec.Index{}, manifests)
	if err != nil {
		return err
	}
	return pushIndex(ctx, target, desc, content, opts.Reference, opts.extraRefs)
}

func resolveSourceManifests(ctx context.Context, target oras.ReadOnlyTarget, sources []string) ([]ocispec.Descriptor, error) {
	var resolved []ocispec.Descriptor
	for _, source := range sources {
		desc, content, err := oras.FetchBytes(ctx, target, source, oras.DefaultFetchBytesOptions)
		if err != nil {
			return nil, err
		}
		if descriptor.IsImageManifest(desc) {
			desc.Platform, err = getPlatform(ctx, target, content)
			if err != nil {
				return nil, err
			}
		}
		resolved = append(resolved, desc)
	}
	return resolved, nil
}

func getPlatform(ctx context.Context, target oras.ReadOnlyTarget, manifestBytes []byte) (*ocispec.Platform, error) {
	// extract config descriptor
	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, err
	}
	// fetch config content
	contentBytes, err := content.FetchAll(ctx, target, manifest.Config)
	if err != nil {
		return nil, err
	}
	var platform ocispec.Platform
	if err := json.Unmarshal(contentBytes, &platform); err != nil {
		return nil, err
	}
	return &platform, nil
}

func packIndex(oldIndex *ocispec.Index, manifests []ocispec.Descriptor) (ocispec.Descriptor, []byte, error) {
	index := ocispec.Index{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		MediaType:    ocispec.MediaTypeImageIndex,
		ArtifactType: oldIndex.ArtifactType,
		Manifests:    manifests,
		Subject:      oldIndex.Subject,
		Annotations:  oldIndex.Annotations,
	}
	indexBytes, err := json.Marshal(index)
	if err != nil {
		return ocispec.Descriptor{}, nil, err
	}
	desc := content.NewDescriptorFromBytes(ocispec.MediaTypeImageIndex, indexBytes)
	return desc, indexBytes, nil
}

func pushIndex(ctx context.Context, target oras.Target, desc ocispec.Descriptor, content []byte, ref string, extraRefs []string) error {
	var err error
	// need to refine the variable names of ref, extra ref
	if ref != "" {
		extraRefs = append(extraRefs, ref)
	}
	if len(extraRefs) == 0 {
		err = target.Push(ctx, desc, bytes.NewReader(content))
	} else {
		desc, err = oras.TagBytesN(ctx, target, desc.MediaType, content, extraRefs, oras.DefaultTagBytesNOptions)
	}
	if err != nil {
		return err
	}
	fmt.Println("Created and pushed index:", desc.Digest)
	return nil
}