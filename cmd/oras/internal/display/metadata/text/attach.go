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

package text

import (
	"fmt"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

// AttachHandler handles text metadata output for attach events.
type AttachHandler struct {
	printer *output.Printer
}

// NewAttachHandler returns a new handler for attach events.
func NewAttachHandler(printer *output.Printer) metadata.AttachHandler {
	return &AttachHandler{
		printer: printer,
	}
}

// OnCompleted is called when the attach command is complete.
func (ah *AttachHandler) OnCompleted(opts *option.Target, root, subject ocispec.Descriptor) error {
	digest := subject.Digest.String()
	if !strings.HasSuffix(opts.RawReference, digest) {
		opts.RawReference = fmt.Sprintf("%s@%s", opts.Path, subject.Digest)
	}
	err := ah.printer.Println("Attached to", opts.AnnotatedReference())
	if err != nil {
		return err
	}
	err = ah.printer.Println("Digest:", root.Digest)
	return err
}
