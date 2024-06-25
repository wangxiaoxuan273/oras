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
	"errors"
	"fmt"
	"io"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/cmd/oras/internal/command"
	"oras.land/oras/cmd/oras/internal/option"
)

type createOptions struct {
	option.Common
	option.Target

	repo string

	sources []option.Target
}

func createCmd() *cobra.Command {
	var opts createOptions
	cmd := &cobra.Command{
		Use:   "create [flags] --repo <repo-reference> <name>[:<tag>|@<digest>] [...]",
		Short: "create an index from provided manifests",
		Long: `create an index to a registry or an OCI image layout
Example - create a index to repository 'localhost:5000/hello':
  oras index create --repo localhost:5000/hello \
     sha256:xxxx \
     sha256:xxxx
`,
		Args: cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// parse the user input
			opts.RawReference = opts.repo
			opts.sources = make([]option.Target, len(args))
			for i, a := range args {
				// assume inputs are tags, TODO digest check, also need to handle OCI layout case
				ref := fmt.Sprintf("%s:%s", opts.repo, a)
				m := option.Target{RawReference: ref}
				if err := m.Parse(cmd); err != nil {
					return err
				}
				opts.sources[i] = m
			}
			return option.Parse(cmd, &opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return createManifest(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.repo, "repo", "", "", "reference of the repository or oci layout")
	_ = cmd.MarkFlagRequired("repo")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func createManifest(cmd *cobra.Command, opts createOptions) error {
	ctx, logger := command.GetLogger(cmd, &opts.Common)

	// Prepare dst target
	dst, err := opts.NewTarget(opts.Common, logger)
	if err != nil {
		return err
	}

	manifests, err := doCopy(cmd, dst, opts, logger)
	if err != nil {
		return err
	}
	if err := doPack(ctx, dst, manifests, opts); err != nil {
		return err
	}
	return nil
}

func doCopy(cmd *cobra.Command, dst oras.GraphTarget, destOpts createOptions, logger logrus.FieldLogger) ([]ocispec.Descriptor, error) {
	baseCopyOptions := oras.DefaultExtendedCopyOptions

	// copy all manifests
	rOpts := oras.DefaultResolveOptions
	var copied []ocispec.Descriptor
	for _, srcOpts := range destOpts.sources {
		var err error
		// prepare src target
		src, err := srcOpts.NewReadonlyTarget(cmd.Context(), destOpts.Common, logger)
		if err != nil {
			return copied, err
		}
		if err := srcOpts.EnsureReferenceNotEmpty(cmd, false); err != nil {
			return nil, err
		}

		copyOptions := baseCopyOptions
		var desc ocispec.Descriptor
		desc, err = oras.Resolve(cmd.Context(), src, srcOpts.Reference, rOpts)
		if err != nil {
			return copied, fmt.Errorf("failed to resolve %s: %w", srcOpts.Reference, err)
		}
		err = oras.CopyGraph(cmd.Context(), src, dst, desc, copyOptions.CopyGraphOptions)
		if err != nil {
			return copied, err
		}
		copied = append(copied, desc)
	}

	return copied, nil
}

func doPack(ctx context.Context, t oras.Target, manifests []ocispec.Descriptor, opts createOptions) error {
	// todo: oras-go needs PackIndex
	index := ocispec.Index{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		MediaType: ocispec.MediaTypeImageIndex,
		Manifests: manifests,
		// todo: annotations
	}
	content, _ := json.Marshal(index)
	reader := bytes.NewReader(content)
	desc := ocispec.Descriptor{
		Digest:    digest.FromBytes(content),
		MediaType: ocispec.MediaTypeImageIndex,
		Size:      int64(len(content)),
	}

	if err := doPushReference(ctx, desc, opts.Reference, t, reader); err != nil {
		return err
	}
	return nil
}

func doPushReference(ctx context.Context, desc ocispec.Descriptor, ref string, dst oras.Target, content io.Reader) error {
	if refPusher, ok := dst.(registry.ReferencePusher); ok {
		if ref != "" {
			return refPusher.PushReference(ctx, desc, content, ref)
		}
	}
	if err := dst.Push(ctx, desc, content); err != nil {
		w := errors.Unwrap(err)
		if w != errdef.ErrAlreadyExists {
			return err
		}
	}
	if ref == "" {
		fmt.Println("Digest of the pushed index: ", desc.Digest)
		return nil
	}
	return dst.Tag(ctx, desc, ref)
}
