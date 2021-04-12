// Copyright 2021 Chaos Mesh Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"github.com/spf13/cobra"

	"github.com/chaos-mesh/chaosd/pkg/core"
)

func commonFlags(cmd *cobra.Command, flag *core.CommonAttackConfig) {
	cmd.Flags().StringVarP(&flag.Schedule, "cron", "", "", "Specify crontab-compatible expression to schedule the attack")
	cmd.Flags().StringVarP(&flag.Duration, "duration", "", "", "Specify how long the experiment run every scheduled run")
}
