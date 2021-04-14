// Copyright 2020 Chaos Mesh Authors.
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
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/chaos-mesh/chaosd/pkg/core"
	"github.com/chaos-mesh/chaosd/pkg/server/chaosd"
)

func NewNetworkAttackCommand() *cobra.Command {
	options := core.NewNetworkCommand()
	dep := fx.Options(
		Module,
		fx.Provide(func() *core.NetworkCommand {
			return options
		}),
	)

	cmd := &cobra.Command{
		Use:   "network <subcommand>",
		Short: "Network attack related commands",
	}

	cmd.AddCommand(
		NewNetworkDelayCommand(dep, options),
		NewNetworkLossCommand(dep, options),
		NewNetworkCorruptCommand(dep, options),
		NetworkDuplicateCommand(dep, options),
	)

	return cmd
}

func NewNetworkDelayCommand(dep fx.Option, options *core.NetworkCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delay",
		Short: "delay network",

		Run: func(*cobra.Command, []string) {
			options.Action = core.NetworkDelayAction
			options.CompleteDefaults()
			fx.New(dep, fx.Invoke(commonNetworkAttackFunc)).Run()
		},
	}

	cmd.Flags().StringVarP(&options.Latency, "latency", "l", "",
		"delay egress time, time units: ns, us (or µs), ms, s, m, h.")
	cmd.Flags().StringVarP(&options.Jitter, "jitter", "j", "",
		"jitter time, time units: ns, us (or µs), ms, s, m, h.")
	cmd.Flags().StringVarP(&options.Correlation, "correlation", "c", "0", "correlation is percentage (10 is 10%)")
	cmd.Flags().StringVarP(&options.Device, "device", "d", "", "the network interface to impact")
	cmd.Flags().StringVarP(&options.EgressPort, "egress-port", "e", "",
		"only impact egress traffic to these destination ports, use a ',' to separate or to indicate the range, such as 80, 8001:8010. "+
			"It can only be used in conjunction with -p tcp or -p udp")
	cmd.Flags().StringVarP(&options.SourcePort, "source-port", "s", "",
		"only impact egress traffic from these source ports, use a ',' to separate or to indicate the range, such as 80, 8001:8010. "+
			"It can only be used in conjunction with -p tcp or -p udp")
	cmd.Flags().StringVarP(&options.IPAddress, "ip", "i", "", "only impact egress traffic to these IP addresses")
	cmd.Flags().StringVarP(&options.Hostname, "hostname", "H", "", "only impact traffic to these hostnames")
	cmd.Flags().StringVarP(&options.IPProtocol, "protocol", "p", "",
		"only impact traffic using this IP protocol, supported: tcp, udp, icmp, all")
	commonFlags(cmd, &options.CommonAttackConfig)

	return cmd
}

func NewNetworkLossCommand(dep fx.Option, options *core.NetworkCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "loss",
		Short: "loss network packet",

		Run: func(*cobra.Command, []string) {
			options.Action = core.NetworkLossAction
			options.CompleteDefaults()
			fx.New(dep, fx.Invoke(commonNetworkAttackFunc)).Run()
		},
	}

	cmd.Flags().StringVar(&options.Percent, "percent", "1", "percentage of packets to drop (10 is 10%)")
	cmd.Flags().StringVarP(&options.Correlation, "correlation", "c", "0", "correlation is percentage (10 is 10%)")
	cmd.Flags().StringVarP(&options.Device, "device", "d", "", "the network interface to impact")
	cmd.Flags().StringVarP(&options.EgressPort, "egress-port", "e", "",
		"only impact egress traffic to these destination ports, use a ',' to separate or to indicate the range, such as 80, 8001:8010. "+
			"It can only be used in conjunction with -p tcp or -p udp")
	cmd.Flags().StringVarP(&options.SourcePort, "source-port", "s", "",
		"only impact egress traffic from these source ports, use a ',' to separate or to indicate the range, such as 80, 8001:8010. "+
			"It can only be used in conjunction with -p tcp or -p udp")
	cmd.Flags().StringVarP(&options.IPAddress, "ip", "i", "", "only impact egress traffic to these IP addresses")
	cmd.Flags().StringVarP(&options.Hostname, "hostname", "H", "", "only impact traffic to these hostnames")
	cmd.Flags().StringVarP(&options.IPProtocol, "protocol", "p", "",
		"only impact traffic using this IP protocol, supported: tcp, udp, icmp, all")
	commonFlags(cmd, &options.CommonAttackConfig)

	return cmd
}

func NewNetworkCorruptCommand(dep fx.Option, options *core.NetworkCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "corrupt",
		Short: "corrupt network packet",

		Run: func(*cobra.Command, []string) {
			options.Action = core.NetworkCorruptAction
			options.CompleteDefaults()
			fx.New(dep, fx.Invoke(commonNetworkAttackFunc)).Run()
		},
	}

	cmd.Flags().StringVar(&options.Percent, "percent", "1", "percentage of packets to corrupt (10 is 10%)")
	cmd.Flags().StringVarP(&options.Correlation, "correlation", "c", "0", "correlation is percentage (10 is 10%)")
	cmd.Flags().StringVarP(&options.Device, "device", "d", "", "the network interface to impact")
	cmd.Flags().StringVarP(&options.EgressPort, "egress-port", "e", "",
		"only impact egress traffic to these destination ports, use a ',' to separate or to indicate the range, such as 80, 8001:8010. "+
			"It can only be used in conjunction with -p tcp or -p udp")
	cmd.Flags().StringVarP(&options.SourcePort, "source-port", "s", "",
		"only impact egress traffic from these source ports, use a ',' to separate or to indicate the range, such as 80, 8001:8010. "+
			"It can only be used in conjunction with -p tcp or -p udp")
	cmd.Flags().StringVarP(&options.IPAddress, "ip", "i", "", "only impact egress traffic to these IP addresses")
	cmd.Flags().StringVarP(&options.Hostname, "hostname", "H", "", "only impact traffic to these hostnames")
	cmd.Flags().StringVarP(&options.IPProtocol, "protocol", "p", "",
		"only impact traffic using this IP protocol, supported: tcp, udp, icmp, all")
	commonFlags(cmd, &options.CommonAttackConfig)

	return cmd
}

func NetworkDuplicateCommand(dep fx.Option, options *core.NetworkCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "duplicate",
		Short: "duplicate network packet",

		Run: func(*cobra.Command, []string) {
			options.Action = core.NetworkDuplicateAction
			options.CompleteDefaults()
			fx.New(dep, fx.Invoke(commonNetworkAttackFunc)).Run()
		},
	}

	cmd.Flags().StringVar(&options.Percent, "percent", "1", "percentage of packets to corrupt (10 is 10%)")
	cmd.Flags().StringVarP(&options.Correlation, "correlation", "c", "0", "correlation is percentage (10 is 10%)")
	cmd.Flags().StringVarP(&options.Device, "device", "d", "", "the network interface to impact")
	cmd.Flags().StringVarP(&options.EgressPort, "egress-port", "e", "",
		"only impact egress traffic to these destination ports, use a ',' to separate or to indicate the range, such as 80, 8001:8010. "+
			"It can only be used in conjunction with -p tcp or -p udp")
	cmd.Flags().StringVarP(&options.SourcePort, "source-port", "s", "",
		"only impact egress traffic from these source ports, use a ',' to separate or to indicate the range, such as 80, 8001:8010. "+
			"It can only be used in conjunction with -p tcp or -p udp")
	cmd.Flags().StringVarP(&options.IPAddress, "ip", "i", "", "only impact egress traffic to these IP addresses")
	cmd.Flags().StringVarP(&options.Hostname, "hostname", "H", "", "only impact traffic to these hostnames")
	cmd.Flags().StringVarP(&options.IPProtocol, "protocol", "p", "",
		"only impact traffic using this IP protocol, supported: tcp, udp, icmp, all")
	commonFlags(cmd, &options.CommonAttackConfig)

	return cmd
}

func commonNetworkAttackFunc(options *core.NetworkCommand, chaos *chaosd.Server) {
	if err := options.Validate(); err != nil {
		ExitWithError(ExitBadArgs, err)
	}

	uid, err := chaos.ExecuteAttack(chaosd.NetworkAttack, options)
	if err != nil {
		ExitWithError(ExitError, err)
	}

	NormalExit(fmt.Sprintf("Attack network successfully, uid: %s", uid))
}
