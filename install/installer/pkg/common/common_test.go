// Copyright (c) 2021 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package common_test

import (
	"testing"

	"github.com/gitpod-io/gitpod/installer/pkg/common"
	"github.com/gitpod-io/gitpod/installer/pkg/config/v1"

	"github.com/google/go-cmp/cmp"
)

func TestRepoName(t *testing.T) {
	type Expectation struct {
		Result string
		Panics bool
	}
	tests := []struct {
		Repo        string
		Name        string
		Expectation Expectation
		CfgRepo     string
	}{
		{
			Name: "gitpod-io/workspace-full",
			Expectation: Expectation{
				Result: "docker.io/gitpod-io/workspace-full",
			},
		},
		{
			Repo: "some-repo.com",
			Name: "some-image",
			Expectation: Expectation{
				Result: "some-repo.com/some-image",
			},
		},
		{
			Repo: "some-repo",
			Name: "not@avalid#image-name",
			Expectation: Expectation{
				Panics: true,
			},
		},
		// Custom repo, no namespace
		{
			Name: "gitpod-io/workspace-full",
			Expectation: Expectation{
				Result: "some.registry.com/workspace-full",
			},
			CfgRepo: "some.registry.com",
		},
		{
			Repo: "some-repo.com",
			Name: "some-image",
			Expectation: Expectation{
				Result: "some.registry.com/some-image",
			},
			CfgRepo: "some.registry.com",
		},
		{
			Repo: "some-repo",
			Name: "not@avalid#image-name",
			Expectation: Expectation{
				Panics: true,
			},
			CfgRepo: "some.registry.com",
		},
		// Custom repo, namespace
		{
			Name: "gitpod-io/workspace-full",
			Expectation: Expectation{
				Result: "some.registry.com/gitpod/workspace-full",
			},
			CfgRepo: "some.registry.com/gitpod",
		},
		{
			Repo: "some-repo.com",
			Name: "some-image",
			Expectation: Expectation{
				Result: "some.registry.com/gitpod/some-image",
			},
			CfgRepo: "some.registry.com/gitpod",
		},
		{
			Repo: "some-repo",
			Name: "not@avalid#image-name",
			Expectation: Expectation{
				Panics: true,
			},
			CfgRepo: "some.registry.com/gitpod",
		},
	}

	for _, test := range tests {
		t.Run(test.Repo+"/"+test.Name, func(t *testing.T) {
			var act Expectation
			func() {
				defer func() {
					if recover() != nil {
						act.Panics = true
					}
				}()
				cfg := config.Config{
					Repository: func() string {
						if test.CfgRepo == "" {
							return common.GitpodContainerRegistry
						}

						return test.CfgRepo
					}(),
				}
				act.Result = common.RepoName(test.Repo, test.Name, &cfg)
			}()

			if diff := cmp.Diff(test.Expectation, act); diff != "" {
				t.Errorf("RepoName() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
