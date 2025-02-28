// Copyright 2023 Upbound Inc
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

package space

import (
	"context"
	"io"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/pterm/pterm"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	"github.com/upbound/up/internal/config"
	"github.com/upbound/up/internal/input"
	"github.com/upbound/up/internal/install"
	"github.com/upbound/up/internal/install/helm"
	"github.com/upbound/up/internal/kube"
	"github.com/upbound/up/internal/upterm"
)

const (
	errParseUpgradeParameters = "unable to parse upgrade parameters"
)

// BeforeApply sets default values in login before assignment and validation.
func (c *upgradeCmd) BeforeApply() error {
	c.prompter = input.NewPrompter()
	return nil
}

// AfterApply sets default values in command after assignment and validation.
func (c *upgradeCmd) AfterApply(insCtx *install.Context, quiet config.QuietFlag) error {
	// NOTE(tnthornton) we currently only have support for stylized output.
	pterm.EnableStyling()
	upterm.DefaultObjPrinter.Pretty = true

	b, err := io.ReadAll(c.TokenFile)
	defer c.TokenFile.Close() // nolint:errcheck
	if err != nil {
		return errors.Wrap(err, errReadTokenFile)
	}
	c.token = string(b)

	c.id = jsonKey
	kClient, err := kubernetes.NewForConfig(insCtx.Kubeconfig)
	if err != nil {
		return err
	}
	c.kClient = kClient
	secret := kube.NewSecretApplicator(kClient)
	c.pullSecret = kube.NewImagePullApplicator(secret)
	ins, err := helm.NewManager(insCtx.Kubeconfig,
		spacesChart,
		c.Repo,
		helm.WithNamespace(ns),
		helm.WithBasicAuth(c.id, c.token),
		helm.IsOCI(),
		helm.WithChart(c.Bundle),
		helm.RollbackOnError(c.Rollback),
		helm.Wait())
	if err != nil {
		return err
	}
	c.helmMgr = ins
	base := map[string]any{}
	if c.File != nil {
		defer c.File.Close() //nolint:errcheck,gosec
		b, err := io.ReadAll(c.File)
		if err != nil {
			return errors.Wrap(err, errReadParametersFile)
		}
		if err := yaml.Unmarshal(b, &base); err != nil {
			return errors.Wrap(err, errReadParametersFile)
		}
		if err := c.File.Close(); err != nil {
			return errors.Wrap(err, errReadParametersFile)
		}
	}
	c.parser = helm.NewParser(base, c.Set)
	c.quiet = quiet
	return nil
}

// upgradeCmd upgrades Upbound.
type upgradeCmd struct {
	helmMgr    install.Manager
	parser     install.ParameterParser
	prompter   input.Prompter
	pullSecret *kube.ImagePullApplicator
	id         string
	token      string
	kClient    kubernetes.Interface
	quiet      config.QuietFlag

	// NOTE(hasheddan): version is currently required for upgrade with OCI image
	// as latest strategy is undetermined.
	Version string `arg:"" help:"Upbound Spaces version to upgrade to."`

	Rollback bool `help:"Rollback to previously installed version on failed upgrade."`

	commonParams
	install.CommonParams
}

// Run executes the upgrade command.
func (c *upgradeCmd) Run(insCtx *install.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	params, err := c.parser.Parse()
	if err != nil {
		return errors.Wrap(err, errParseUpgradeParameters)
	}

	// Create or update image pull secret.
	if err := c.pullSecret.Apply(ctx, defaultImagePullSecret, ns, c.id, c.token, c.Registry.String()); err != nil {
		return errors.Wrap(err, errCreateImagePullSecret)
	}

	if err := c.upgradeUpbound(params); err != nil {
		return err
	}

	return nil
}

func (c *upgradeCmd) upgradeUpbound(params map[string]any) error {
	upgrade := func() error {
		if err := c.helmMgr.Upgrade(strings.TrimPrefix(c.Version, "v"), params); err != nil {
			return err
		}
		return nil
	}

	if err := upterm.WrapWithSuccessSpinner(
		"Upgrading Space",
		upterm.CheckmarkSuccessSpinner,
		upgrade,
	); err != nil {
		return err
	}

	return nil
}
