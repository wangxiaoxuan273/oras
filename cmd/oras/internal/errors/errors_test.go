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

package errors

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/spf13/pflag"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote/errcode"
)

func TestCheckMutuallyExclusiveFlags(t *testing.T) {
	fs := &pflag.FlagSet{}
	var foo, bar, hello bool
	fs.BoolVar(&foo, "foo", false, "foo test")
	fs.BoolVar(&bar, "bar", false, "bar test")
	fs.BoolVar(&hello, "hello", false, "hello test")
	fs.Lookup("foo").Changed = true
	fs.Lookup("bar").Changed = true
	tests := []struct {
		name             string
		exclusiveFlagSet []string
		wantErr          bool
	}{
		{
			"--foo and --bar should not be used at the same time",
			[]string{"foo", "bar"},
			true,
		},
		{
			"--foo and --hello are not used at the same time",
			[]string{"foo", "hello"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckMutuallyExclusiveFlags(fs, tt.exclusiveFlagSet...); (err != nil) != tt.wantErr {
				t.Errorf("CheckMutuallyExclusiveFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckRequiredTogetherFlags(t *testing.T) {
	fs := &pflag.FlagSet{}
	var foo, bar, hello, world bool
	fs.BoolVar(&foo, "foo", false, "foo test")
	fs.BoolVar(&bar, "bar", false, "bar test")
	fs.BoolVar(&hello, "hello", false, "hello test")
	fs.BoolVar(&world, "world", false, "world test")
	fs.Lookup("foo").Changed = true
	fs.Lookup("bar").Changed = true
	tests := []struct {
		name                  string
		requiredTogetherFlags []string
		wantErr               bool
	}{
		{
			"--foo and --bar are both used, no error is returned",
			[]string{"foo", "bar"},
			false,
		},
		{
			"--foo and --hello are not both used, an error is returned",
			[]string{"foo", "hello"},
			true,
		},
		{
			"none of --hello and --world is used, no error is returned",
			[]string{"hello", "world"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckRequiredTogetherFlags(fs, tt.requiredTogetherFlags...); (err != nil) != tt.wantErr {
				t.Errorf("CheckRequiredTogetherFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReportErrResp(t *testing.T) {
	// Test case with empty errors
	emptyErrorsResp := &errcode.ErrorResponse{
		Errors:     []errcode.Error{},
		StatusCode: 401,
		URL:        &url.URL{Host: "localhost:5000"},
		Method:     "GET",
	}

	// Test case with non-empty errors
	nonEmptyErrorsResp := &errcode.ErrorResponse{
		Errors: []errcode.Error{
			{
				Code:    "UNAUTHORIZED",
				Message: "authentication required",
			},
			{
				Code:    "INVALID_CREDENTIALS",
				Message: "invalid credentials provided",
				Detail:  "please check your username and password",
			},
		},
		StatusCode: 401,
		URL:        &url.URL{Host: "localhost:5000"},
		Method:     "GET",
	}

	tests := []struct {
		name    string
		errResp *errcode.ErrorResponse
		wantErr error
	}{
		{
			name:    "empty errors",
			errResp: emptyErrorsResp,
			wantErr: emptyErrorsResp,
		},
		{
			name:    "non-empty errors",
			errResp: nonEmptyErrorsResp,
			wantErr: nonEmptyErrorsResp.Errors,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReportErrResp(tt.errResp)
			if got.Error() != tt.wantErr.Error() {
				t.Errorf("ReportErrResp() = %v, want %v", got, tt.wantErr)
			}
		})
	}
}

func TestUnwrapCopyError(t *testing.T) {
	// Create a regular error
	regularErr := fmt.Errorf("regular error")

	// Create an oras.CopyError with an inner error
	innerErr := fmt.Errorf("inner error")
	copyErr := &oras.CopyError{Err: innerErr}

	tests := []struct {
		name     string
		inputErr error
		wantErr  error
	}{
		{
			name:     "nil error",
			inputErr: nil,
			wantErr:  nil,
		},
		{
			name:     "regular error",
			inputErr: regularErr,
			wantErr:  regularErr,
		},
		{
			name:     "copy error",
			inputErr: copyErr,
			wantErr:  innerErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := UnwrapCopyError(tt.inputErr)
			if gotErr != tt.wantErr {
				t.Errorf("UnwrapCopyError() = %v, want %v", gotErr, tt.wantErr)
			}
		})
	}
}
