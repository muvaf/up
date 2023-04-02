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

	"github.com/alecthomas/kong"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/posener/complete"

	"github.com/upbound/up-sdk-go/service/accounts"
	"github.com/upbound/up-sdk-go/service/organizations"
	"github.com/upbound/up-sdk-go/service/robots"

	"github.com/upbound/up/cmd/up/robot/token"
	"github.com/upbound/up/internal/upbound"
)

const (
	errUserAccountFmt = "%s is a user account but robot operations are supported only for organization accounts. use the --account flag to specify an organization account."
)

// AfterApply constructs and binds a robots client to any subcommands
// that have Run() methods that receive it.
func (c *Cmd) AfterApply(kongCtx *kong.Context) error {
	upCtx, err := upbound.NewFromFlags(c.Flags)
	if err != nil {
		return err
	}
	// There is no way for users to own any control planes.
	if upCtx.Account.Type == accounts.AccountUser {
		return errors.Errorf(errUserAccountFmt, upCtx.Account.Name)
	}
	cfg, err := upCtx.BuildSDKConfig()
	if err != nil {
		return err
	}
	kongCtx.Bind(upCtx)
	kongCtx.Bind(accounts.NewClient(cfg))
	kongCtx.Bind(organizations.NewClient(cfg))
	kongCtx.Bind(robots.NewClient(cfg))
	return nil
}

func PredictRobots() complete.Predictor {
	return complete.PredictFunc(func(a complete.Args) (prediction []string) {
		upCtx, err := upbound.NewFromFlags(upbound.Flags{})
		if err != nil {
			return nil
		}
		cfg, err := upCtx.BuildSDKConfig()
		if err != nil {
			return nil
		}

		ac := accounts.NewClient(cfg)
		if ac == nil {
			return nil
		}

		oc := organizations.NewClient(cfg)
		if oc == nil {
			return nil
		}

		account, err := ac.Get(context.Background(), upCtx.Account.Name)
		if err != nil {
			return nil
		}
		if account.Account.Type != accounts.AccountOrganization {
			return nil
		}
		rs, err := oc.ListRobots(context.Background(), account.Organization.ID)
		if err != nil {
			return nil
		}
		if len(rs) == 0 {
			return nil
		}
		data := make([]string, len(rs))
		for i, r := range rs {
			data[i] = r.Name
		}
		return data
	})
}

// Cmd contains commands for interacting with robots.
type Cmd struct {
	Create createCmd `cmd:"" help:"Create a robot."`
	Delete deleteCmd `cmd:"" help:"Delete a robot."`
	List   listCmd   `cmd:"" help:"List robots for the account."`
	Get    getCmd    `cmd:"" help:"Get a robot for the account."`
	Token  token.Cmd `cmd:"" help:"Interact with robot tokens."`

	// Common Upbound API configuration
	Flags upbound.Flags `embed:""`
}
