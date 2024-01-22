// Copyright 2021 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package tests

import (
	"context"
	"regexp"

	"github.com/cockroachdb/cockroach/pkg/cmd/roachtest/cluster"
	"github.com/cockroachdb/cockroach/pkg/cmd/roachtest/option"
	"github.com/cockroachdb/cockroach/pkg/cmd/roachtest/registry"
	"github.com/cockroachdb/cockroach/pkg/cmd/roachtest/spec"
	"github.com/cockroachdb/cockroach/pkg/cmd/roachtest/test"
	"github.com/cockroachdb/cockroach/pkg/roachprod/install"
)

const asyncpgRunTestCmd = `
source venv/bin/activate && 
cd /mnt/data1/asyncpg && 
PGPORT={pgport:1} PGHOST=localhost PGUSER=root PGDATABASE=defaultdb python3 setup.py test > asyncpg.stdout
`

var asyncpgReleaseTagRegex = regexp.MustCompile(`^(?P<major>v\d+)\.(?P<minor>\d+)\.(?P<point>\d+)$`)

// WARNING: DO NOT MODIFY the name of the below constant/variable without approval from the docs team.
// This is used by docs automation to produce a list of supported versions for ORM's.
var asyncpgSupportedTag = "v0.24.0"

func registerAsyncpg(r registry.Registry) {
	runAsyncpg := func(
		ctx context.Context,
		t test.Test,
		c cluster.Cluster,
	) {
		if c.IsLocal() {
			t.Fatal("cannot be run in local mode")
		}
		node := c.Node(1)
		t.Status("setting up cockroach")

		// This test assumes that multiple_active_portals_enabled is false, but through
		// metamorphic constants, it is possible for them to be enabled. We disable
		// metamorphic testing to avoid this. Note the asyncpg test suite drops the
		// database so we can't set the session variable like we do in pgjdbc.
		// TODO(DarrylWong): Use a metamorphic constants exclusion list instead.
		// See: https://github.com/cockroachdb/cockroach/issues/113164
		settings := install.MakeClusterSettings()
		settings.Env = append(settings.Env, "COCKROACH_INTERNAL_DISABLE_METAMORPHIC_TESTING=true")
		c.Start(ctx, t.L(), option.DefaultStartOptsInMemory(), settings, c.All())

		version, err := fetchCockroachVersion(ctx, t.L(), c, node[0])
		if err != nil {
			t.Fatal(err)
		}

		if err := alterZoneConfigAndClusterSettings(ctx, t, version, c, node[0]); err != nil {
			t.Fatal(err)
		}

		t.Status("cloning asyncpg and installing prerequisites")

		latestTag, err := repeatGetLatestTag(
			ctx, t, "MagicStack", "asyncpg", asyncpgReleaseTagRegex,
		)
		if err != nil {
			t.Fatal(err)
		}

		t.L().Printf("Latest asyncpg release is %s.", latestTag)
		t.L().Printf("Supported asyncpg release is %s.", latestTag)

		if err := gitCloneWithRecurseSubmodules(
			ctx,
			c,
			t.L(),
			"https://github.com/MagicStack/asyncpg.git",
			"/mnt/data1/asyncpg",
			asyncpgSupportedTag,
			node,
		); err != nil {
			t.Fatal(err)
		}

		if err := repeatRunE(
			ctx, t, c, node, "update apt-get",
			`
				sudo add-apt-repository ppa:deadsnakes/ppa &&
				sudo apt-get -qq update`,
		); err != nil {
			t.Fatal(err)
		}

		if err := repeatRunE(
			ctx,
			t,
			c,
			node,
			"install python and pip",
			`sudo apt-get -qq install python3.8 python3-pip libpq-dev python3.8-dev python3-virtualenv python3.8-distutils python3-apt python3-setuptools python-setuptools`,
		); err != nil {
			t.Fatal(err)
		}

		if err := repeatRunE(
			ctx, t, c, node, "set python3.8 as default", `
    		sudo update-alternatives --install /usr/bin/python3 python3 /usr/bin/python3.5 1
    		sudo update-alternatives --install /usr/bin/python3 python3 /usr/bin/python3.8 2
    		sudo update-alternatives --config python3`,
		); err != nil {
			t.Fatal(err)
		}

		if err := repeatRunE(
			ctx, t, c, node, "install pip",
			`curl https://bootstrap.pypa.io/get-pip.py | sudo -H python3.8`,
		); err != nil {
			t.Fatal(err)
		}

		if err := repeatRunE(
			ctx, t, c, node, "create virtualenv", `virtualenv --clear venv`,
		); err != nil {
			t.Fatal(err)
		}

		if err := repeatRunE(
			ctx,
			t,
			c,
			node,
			"install asyncpg's dependencies",
			"source venv/bin/activate && cd /mnt/data1/asyncpg && pip3 install -e ."); err != nil {
			t.Fatal(err)
		}

		blocklistName, expectedFailureList := "asyncpgBlocklist", asyncpgBlocklist
		ignoredlistName, ignoredlist := "asyncpgIgnoreList", asyncpgIgnoreList
		if ignoredlist == nil {
			t.Fatalf("No asyncpg ignorelist defined for cockroach version %s", version)
		}
		t.L().Printf("Running cockroach version %s, using blocklist %s, using ignorelist %s",
			version, blocklistName, ignoredlistName)

		t.Status("Running asyncpg tests ")
		result, err := c.RunWithDetailsSingleNode(
			ctx, t.L(), option.WithNodes(node), asyncpgRunTestCmd)
		if err != nil {
			t.L().Printf("error during asyncpg run (may be ok): %v\n", err)
		}
		t.L().Printf("Test results for asyncpg: %s", result.Stdout+result.Stderr)
		t.L().Printf("Test stdout for asyncpg")
		if err := c.RunE(
			ctx, option.WithNodes(node), "cd /mnt/data1/asyncpg && cat asyncpg.stdout",
		); err != nil {
			t.Fatal(err)
		}

		t.Status("collating test results")

		results := newORMTestsResults()
		results.parsePythonUnitTestOutput([]byte(result.Stdout+result.Stderr), expectedFailureList, ignoredlist)
		results.summarizeAll(
			t, "asyncpg" /* ormName */, blocklistName, expectedFailureList, version, asyncpgSupportedTag,
		)
	}

	r.Add(registry.TestSpec{
		Name:             "asyncpg",
		Owner:            registry.OwnerSQLFoundations,
		Cluster:          r.MakeClusterSpec(1, spec.CPU(16)),
		CompatibleClouds: registry.AllExceptAWS,
		Suites:           registry.Suites(registry.Nightly, registry.ORM),
		Leases:           registry.MetamorphicLeases,
		Run: func(ctx context.Context, t test.Test, c cluster.Cluster) {
			runAsyncpg(ctx, t, c)
		},
	})
}
