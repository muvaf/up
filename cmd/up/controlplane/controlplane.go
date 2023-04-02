// Copyright 2021 Upbound Inc
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

package controlplane

import (
	"context"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/upbound/up-sdk-go/service/accounts"

	"github.com/alecthomas/kong"
	"github.com/posener/complete"

	"github.com/upbound/up-sdk-go/service/configurations"
	cp "github.com/upbound/up-sdk-go/service/controlplanes"
	"github.com/upbound/up/cmd/up/controlplane/kubeconfig"
	"github.com/upbound/up/cmd/up/controlplane/pkg"
	"github.com/upbound/up/cmd/up/controlplane/pullsecret"
	"github.com/upbound/up/internal/feature"
	"github.com/upbound/up/internal/upbound"
)

const (
	errUserAccountFmt = "%s is a user account but control plane operations are supported only for organization accounts. use the --account flag to specify an organization account."
)

// BeforeReset is the first hook to run.
func (c *Cmd) BeforeReset(p *kong.Path, maturity feature.Maturity) error {
	return feature.HideMaturity(p, maturity)
}

// AfterApply constructs and binds a control plane client to any subcommands
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
	kongCtx.Bind(cp.NewClient(cfg))
	kongCtx.Bind(configurations.NewClient(cfg))
	return nil
}

func PredictControlPlanes() complete.Predictor {
	return complete.PredictFunc(func(a complete.Args) (prediction []string) {
		upCtx, err := upbound.NewFromFlags(upbound.Flags{})
		if err != nil {
			return nil
		}
		cfg, err := upCtx.BuildSDKConfig()
		if err != nil {
			return nil
		}

		cp := cp.NewClient(cfg)
		if cp == nil {
			return nil
		}

		ctps, err := cp.List(context.Background(), upCtx.Account.Name)
		if err != nil {
			return nil
		}

		if len(ctps.ControlPlanes) == 0 {
			return nil
		}

		data := make([]string, len(ctps.ControlPlanes))
		for i, ctp := range ctps.ControlPlanes {
			data[i] = ctp.ControlPlane.Name
		}
		return data
	})
}

// Cmd contains commands for interacting with control planes.
type Cmd struct {
	Create createCmd `cmd:"" help:"Create a managed control plane."`
	Delete deleteCmd `cmd:"" help:"Delete a control plane."`
	List   listCmd   `cmd:"" help:"List control planes for the account."`
	Get    getCmd    `cmd:"" help:"Get a single control plane."`

	Connect connectCmd `cmd:"" help:"Connect an App Cluster to a managed control plane."`

	Configuration pkg.Cmd `cmd:"" set:"package_type=Configuration" help:"Manage Configurations."`
	Provider      pkg.Cmd `cmd:"" set:"package_type=Provider" help:"Manage Providers."`

	PullSecret pullsecret.Cmd `cmd:"" help:"Manage package pull secrets."`

	Kubeconfig kubeconfig.Cmd `cmd:"" name:"kubeconfig" help:"Manage control plane kubeconfig data."`

	// Common Upbound API configuration
	Flags upbound.Flags `embed:""`
}
