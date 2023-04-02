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

package token

import (
	"context"
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/google/uuid"
	"github.com/pterm/pterm"

	"github.com/upbound/up-sdk-go/service/organizations"
	"github.com/upbound/up-sdk-go/service/robots"
	"github.com/upbound/up-sdk-go/service/tokens"

	"github.com/upbound/up/internal/input"
	"github.com/upbound/up/internal/upbound"
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

	confirm, err := c.prompter.Prompt("Are you sure you want to delete this robot token? [y/n]", false)
	if err != nil {
		return err
	}

	if input.InputYes(confirm) {
		p.Printfln("Deleting robot token %s/%s/%s. This cannot be undone.", upCtx.Account.Name, c.RobotName, c.TokenName)
		return nil
	}

	return fmt.Errorf("operation canceled")
}

// deleteCmd deletes a robot token on Upbound.
type deleteCmd struct {
	prompter input.Prompter

	RobotName string `arg:"" required:"" help:"Name of robot."`
	TokenName string `arg:"" required:"" help:"Name of token."`

	Force bool `help:"Force delete token even if conflicts exist." default:"false"`
}

// Run executes the delete command.
func (c *deleteCmd) Run(p pterm.TextPrinter, oc *organizations.Client, rc *robots.Client, tc *tokens.Client, upCtx *upbound.Context) error { //nolint:gocyclo
	rs, err := oc.ListRobots(context.Background(), upCtx.Account.ID)
	if err != nil {
		return err
	}
	if len(rs) == 0 {
		return errors.Errorf(errFindRobotFmt, c.RobotName, upCtx.Account.Name)
	}
	// TODO(hasheddan): because this API does not guarantee name uniqueness, we
	// must guarantee that exactly one robot exists in the specified account
	// with the provided name. Logic should be simplified when the API is
	// updated.
	var rid *uuid.UUID
	for _, r := range rs {
		if r.Name == c.RobotName {
			if rid != nil {
				return errors.Errorf(errMultipleRobotFmt, c.RobotName, upCtx.Account.Name)
			}
			// Pin range variable so that we can take address.
			r := r
			rid = &r.ID
		}
	}
	if rid == nil {
		return errors.Errorf(errFindRobotFmt, c.RobotName, upCtx.Account.Name)
	}

	ts, err := rc.ListTokens(context.Background(), *rid)
	if err != nil {
		return err
	}
	if len(ts.DataSet) == 0 {
		return errors.Errorf(errFindTokenFmt, c.TokenName, c.RobotName, upCtx.Account.Name)
	}

	// TODO(hasheddan): because this API does not guarantee name uniqueness, we
	// must guarantee that exactly one token exists for the specified robot in
	// the specified account with the provided name. Logic should be simplified
	// when the API is updated.
	var tid *uuid.UUID
	for _, t := range ts.DataSet {
		if fmt.Sprint(t.AttributeSet["name"]) == c.TokenName {
			if tid != nil && !c.Force {
				return errors.Errorf(errMultipleTokenFmt, c.TokenName, c.RobotName, upCtx.Account.Name)
			}
			// Pin range variable so that we can take address.
			t := t
			tid = &t.ID
		}
	}
	if tid == nil {
		return errors.Errorf(errFindTokenFmt, c.TokenName, c.RobotName, upCtx.Account.Name)
	}

	if err := tc.Delete(context.Background(), *tid); err != nil {
		return err
	}
	p.Printfln("%s/%s/%s deleted", upCtx.Account.Name, c.RobotName, c.TokenName)
	return nil
}
