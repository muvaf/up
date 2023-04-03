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

	"github.com/pterm/pterm"

	"github.com/upbound/up-sdk-go"
	"github.com/upbound/up-sdk-go/service/accounts"
	"github.com/upbound/up-sdk-go/service/controlplanes"
)

// deleteCmd deletes a control plane on Upbound.
type deleteCmd struct {
	Name string `arg:"" help:"Name of control plane." predictor:"ctps"`
}

// Run executes the delete command.
func (c *deleteCmd) Run(p pterm.TextPrinter, a *accounts.AccountResponse, cfg *up.Config) error {
	if err := controlplanes.NewClient(cfg).Delete(context.Background(), a.Account.Name, c.Name); err != nil {
		return err
	}
	p.Printfln("%s deleted", c.Name)
	return nil
}
