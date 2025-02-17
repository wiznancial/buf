// Copyright 2020-2021 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bufcli

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/rpc"
)

var (
	// ErrNoModuleName is used when the user does not specify a module name in their configuration file.
	ErrNoModuleName = errors.New(`please specify a module name in your configuration file with the "name" key`)

	// ErrNoConfigFile is used when the user tries to execute a command without a configuration file.
	ErrNoConfigFile = errors.New(`please define a configuration file in the current directory; you can create one by running "buf beta mod init"`)
)

// errInternal is returned when the user encounters an unexpected internal buf error.
type errInternal struct {
	cause error
}

// NewInternalError represents an internal error encountered by the buf CLI.
// These errors should not happen and therefore warrant a bug report.
func NewInternalError(err error) error {
	if isInternalError(err) {
		return err
	}
	return &errInternal{cause: err}
}

// isInternalError returns whether the error provided, or
// any error wrapped by that error, is an internal error.
func isInternalError(err error) bool {
	return errors.Is(err, &errInternal{})
}

func (e *errInternal) Error() string {
	message := "it looks like you have found a bug in buf. " +
		"Please file an issue at https://github.com/bufbuild/buf/issues/ " +
		"and provide the command you ran"
	if e.cause == nil {
		return message
	}
	return message + ", as well as the following message: " + e.cause.Error()
}

// Is implements errors.Is for errInternal.
func (e *errInternal) Is(err error) bool {
	_, ok := err.(*errInternal)
	return ok
}

// NewErrorInterceptor returns a CLI interceptor that wraps Buf CLI errors.
func NewErrorInterceptor(action string) appflag.Interceptor {
	return func(next func(context.Context, appflag.Container) error) func(context.Context, appflag.Container) error {
		return func(ctx context.Context, container appflag.Container) error {
			return wrapError(action, next(ctx, container))
		}
	}
}

// NewModuleRefError is used when the client fails to parse a module ref.
func NewModuleRefError(moduleRef string) error {
	return fmt.Errorf("could not parse %q as a module; please verify this is a valid reference", moduleRef)
}

// NewTooManyEmptyAnswersError is used when the user does not answer a prompt in
// the given number of attempts.
func NewTooManyEmptyAnswersError(attempts int) error {
	return fmt.Errorf("did not receive an answer in %d attempts", attempts)
}

// NewFlagIsRequiredError informs the user that a given flag is required.
func NewFlagIsRequiredError(flagName string) error {
	return appcmd.NewInvalidArgumentErrorf("--%s is required.", flagName)
}

// NewOrganizationNameAlreadyExistsError informs the user that an organization with
// that name already exists.
func NewOrganizationNameAlreadyExistsError(name string) error {
	return fmt.Errorf("an organization named %q already exists", name)
}

// NewRepositoryNameAlreadyExistsError informs the user that a repository
// with that name already exists.
func NewRepositoryNameAlreadyExistsError(name string) error {
	return fmt.Errorf("a repository named %q already exists", name)
}

// NewBranchNameAlreadyExistsError informs the user that a branch
// with that name already exists.
func NewBranchNameAlreadyExistsError(name string) error {
	return fmt.Errorf("a branch named %q already exists", name)
}

// NewOrganizationNotFoundError informs the user that an organization with
// that name does not exist.
func NewOrganizationNotFoundError(name string) error {
	return fmt.Errorf(`an organization named %q does not exist, use "buf beta registry organization create" to create one`, name)
}

// NewRepositoryNotFoundError informs the user that a repository with
// that name does not exist.
func NewRepositoryNotFoundError(name string) error {
	return fmt.Errorf(`a repository named %q does not exist, use "buf beta registry repository create" to create one`, name)
}

// NewTokenNotFoundError informs the user that a token with
// that identifier does not exist.
func NewTokenNotFoundError(tokenID string) error {
	return fmt.Errorf("a token with ID %q does not exist", tokenID)
}

// wrapError is used when a CLI command fails, regardless of its error code.
// Note that this function will wrap the error so that the underlying error
// can be recovered via 'errors.Is'.
func wrapError(action string, err error) error {
	if err == nil || (err.Error() == "" && rpc.GetErrorCode(err) == rpc.ErrorCodeUnknown) {
		// If the error is nil or empty and not an rpc error, we return it as-is.
		// This is especially relevant for commands like lint and breaking.
		return err
	}
	switch {
	case rpc.GetErrorCode(err) == rpc.ErrorCodeUnauthenticated:
		return fmt.Errorf(`Failed to %s; you are not authenticated. Create a new entry in your netrc, using a Buf API Key as the password. For details, visit https://beta.docs.buf.build/authentication`, action)
	case rpc.GetErrorCode(err) == rpc.ErrorCodeUnavailable:
		return fmt.Errorf(`Failed to %s: the server hosted at that remote is unavailable: %w.`, action, err)
	}
	return fmt.Errorf("Failed to %q: %w.", action, err)
}
