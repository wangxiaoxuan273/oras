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

package content

import ocispec "github.com/opencontainers/image-spec/specs-go/v1"

type DiscardHandler struct{}

// OnContentFetched implements ManifestFetchHandler.
func (DiscardHandler) OnContentFetched(ocispec.Descriptor, []byte) error {
	return nil
}

// OnContentCreated implements ManifestIndexCreateHandler.
func (DiscardHandler) OnContentCreated([]byte) error {
	return nil
}

// OnBlobPushed implements BlobPushHandler.
func (DiscardHandler) OnBlobPushed() error {
	return nil
}

// NewDiscardHandler returns a new discard handler.
func NewDiscardHandler() DiscardHandler {
	return DiscardHandler{}
}
