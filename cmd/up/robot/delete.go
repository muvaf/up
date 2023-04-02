// Copyright 2022 Upbound Inc
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

package robot

import (
	"context"
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/google/uuid"
	"github.com/pterm/pterm"

	"github.com/upbound/up-sdk-go/service/organizations"
	"github.com/upbound/up-sdk-go/service/robots"

	"github.com/upbound/up/internal/input"
	"github.com/upbound/up/internal/upbound"
)

const (
	errMultipleRobotFmt = "found multiple robots with name %s in %s"
	errFindRobotFmt     = "could not find robot %s in %s"
)

// BeforeApply sets default values for the delete command, before assignment and validation.
func (c *deleteCmd) BeforeApply() error {
	c.prompter = input.NewPrompter()
	return nil
}

// AfterApply accepts user input by default to confirm the delete operation.
func (c *deleteCmd) AfterApply(p pterm.TextPrinter, upCtx *upbound.Context) error {
	if c.Force {
		return nil
	}

	confirm, err := c.prompter.Prompt("Are you sure you want to delete this robot? [y/n]", false)
	if err != nil {
		return err
	}

	if input.InputYes(confirm) {
		p.Printfln("Deleting robot %s/%s. This cannot be undone.", upCtx.Account.Name, c.Name)
		return nil
	}

	return fmt.Errorf("operation canceled")
}

// deleteCmd deletes a robot on Upbound.
type deleteCmd struct {
	prompter input.Prompter

	Name string `arg:"" required:"" help:"Name of robot." predictor:"robots"`

	Force bool `help:"Force delete robot even if conflicts exist." default:"false"`
}

// Run executes the delete command.
func (c *deleteCmd) Run(p pterm.TextPrinter, oc *organizations.Client, rc *robots.Client, upCtx *upbound.Context) error { //nolint:gocyclo
	rs, err := oc.ListRobots(context.Background(), upCtx.Account.ID)
	if err != nil {
		return err
	}
	if len(rs) == 0 {
		return errors.Errorf(errFindRobotFmt, c.Name, upCtx.Account.Name)
	}
	// TODO(hasheddan): because this API does not guarantee name uniqueness, we
	// must guarantee that exactly one robot exists in the specified account
	// with the provided name. Logic should be simplified when the API is
	// updated.
	var id *uuid.UUID
	for _, r := range rs {
		if r.Name == c.Name {
			if id != nil && !c.Force {
				return errors.Errorf(errMultipleRobotFmt, c.Name, upCtx.Account.Name)
			}
			// Pin range variable so that we can take address.
			r := r
			id = &r.ID
		}
	}

	if id == nil {
		return errors.Errorf(errFindRobotFmt, c.Name, upCtx.Account.Name)
	}

	if err := rc.Delete(context.Background(), *id); err != nil {
		return err
	}
	p.Printfln("%s/%s deleted", upCtx.Account.Name, c.Name)
	return nil
}
