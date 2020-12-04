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

package chaosd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/google/uuid"
	"github.com/pingcap/errors"
	"github.com/pingcap/log"

	"github.com/chaos-mesh/chaos-mesh/pkg/chaosdaemon/pb"

	"github.com/chaos-mesh/chaosd/pkg/core"
)

const (
	NetworkAttack = "network attack"
)

func (s *Server) NetworkAttack(attack *core.NetworkCommand) (string, error) {
	var (
		ipsetName string
		err       error
	)
	uid := uuid.New().String()

	if err = s.exp.Set(context.Background(), &core.Experiment{
		Uid:            uid,
		Status:         core.Created,
		Kind:           NetworkAttack,
		RecoverCommand: attack.String(),
	}); err != nil {
		return "", errors.WithStack(err)
	}

	defer func() {
		if err != nil {
			if err := s.exp.Update(context.Background(), uid, core.Error, err.Error(), attack.String()); err != nil {
				log.Error("failed to update experiment", zap.Error(err))
			}
			return
		}
		if err := s.exp.Update(context.Background(), uid, core.Success, "", attack.String()); err != nil {
			log.Error("failed to update experiment", zap.Error(err))
		}
	}()

	if attack.NeedApplyIPSet() {
		ipsetName, err = s.applyIPSet(attack, uid)
		if err != nil {
			return "", errors.WithStack(err)
		}
	}

	if attack.NeedApplyIptables() {
		if err = s.applyIptables(attack, uid); err != nil {
			return "", errors.WithStack(err)
		}
	}

	if attack.NeedApplyTC() {
		if err = s.applyTC(attack, ipsetName, uid); err != nil {
			return "", errors.WithStack(err)
		}
	}

	if err = s.exp.Update(context.Background(), uid, core.Success, "", attack.String()); err != nil {
		return "", errors.WithStack(err)
	}

	return uid, nil
}

func (s *Server) applyIPSet(attack *core.NetworkCommand, uid string) (string, error) {
	ipset, err := attack.ToIPSet(fmt.Sprintf("chaos-%s", uid[:16]))
	if err != nil {
		return "", errors.WithStack(err)
	}

	if _, err := s.svr.FlushIPSets(context.Background(), &pb.IPSetsRequest{
		Ipsets: []*pb.IPSet{ipset},
	}); err != nil {
		return "", errors.WithStack(err)
	}

	if err := s.ipsetRule.Set(context.Background(), &core.IPSetRule{
		Name:       ipset.Name,
		Cidrs:      strings.Join(ipset.Cidrs, ","),
		Experiment: uid,
	}); err != nil {
		return "", errors.WithStack(err)
	}

	return ipset.Name, nil
}

func (s *Server) applyIptables(attack *core.NetworkCommand, uid string) error {
	iptables, err := s.iptablesRule.List(context.Background())
	if err != nil {
		return errors.WithStack(err)
	}
	chains := core.IptablesRuleList(iptables).ToChains()
	newChain, err := attack.ToChain()
	if err != nil {
		return errors.WithStack(err)
	}

	chains = append(chains, newChain)
	if _, err := s.svr.SetIptablesChains(context.Background(), &pb.IptablesChainsRequest{
		Chains: chains,
	}); err != nil {
		return errors.WithStack(err)
	}

	if err := s.iptablesRule.Set(context.Background(), &core.IptablesRule{
		Name:       newChain.Name,
		IPSets:     strings.Join(newChain.Ipsets, ","),
		Direction:  pb.Chain_Direction_name[int32(newChain.Direction)],
		Experiment: uid,
	}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (s *Server) applyTC(attack *core.NetworkCommand, ipset string, uid string) error {
	tcRules, err := s.tcRule.FindByDevice(context.Background(), attack.Device)
	if err != nil {
		return errors.WithStack(err)
	}

	tcs, err := core.TCRuleList(tcRules).ToTCs()
	if err != nil {
		return errors.WithStack(err)
	}

	newTC, err := attack.ToTC(ipset)
	if err != nil {
		return errors.WithStack(err)
	}

	tcs = append(tcs, newTC)
	if _, err := s.svr.SetTcs(context.Background(), &pb.TcsRequest{Tcs: tcs, Device: attack.Device}); err != nil {
		return errors.WithStack(err)
	}

	tc := &core.TcParameter{
		Device: attack.Device,
	}
	switch attack.Action {
	case core.NetworkDelayAction:
		tc.Delay = &core.DelaySpec{
			Latency:     attack.Latency,
			Correlation: attack.Correlation,
			Jitter:      attack.Jitter,
		}
	case core.NetworkLossAction:
		tc.Loss = &core.LossSpec{
			Loss:        attack.Percent,
			Correlation: attack.Correlation,
		}
	default:
		return errors.Errorf("network %s attack not supported", attack.Action)
	}

	tcString, err := json.Marshal(tc)
	if err != nil {
		return errors.WithStack(err)
	}

	if err := s.tcRule.Set(context.Background(), &core.TCRule{
		Type:       pb.Tc_Type_name[int32(newTC.Type)],
		Device:     attack.Device,
		TC:         string(tcString),
		IPSet:      newTC.Ipset,
		Protocal:   newTC.Protocol,
		SourcePort: newTC.SourcePort,
		EgressPort: newTC.EgressPort,
		Experiment: uid,
	}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (s *Server) RecoverNetworkAttack(uid string, attack *core.NetworkCommand) error {
	if attack.NeedApplyIPSet() {
		if err := s.recoverIPSet(uid); err != nil {
			return errors.WithStack(err)
		}
	}

	if attack.NeedApplyIptables() {
		if err := s.recoverIptables(uid); err != nil {
			return errors.WithStack(err)
		}
	}

	if attack.NeedApplyTC() {
		if err := s.recoverTC(uid, attack.Device); err != nil {
			return errors.WithStack(err)
		}
	}

	return errors.WithStack(s.exp.Update(context.Background(),
		uid, core.Destroyed, "", attack.String()))
}

func (s *Server) recoverIPSet(uid string) error {
	if err := s.ipsetRule.DeleteByExperiment(context.Background(), uid); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (s *Server) recoverIptables(uid string) error {
	if err := s.iptablesRule.DeleteByExperiment(context.Background(), uid); err != nil {
		return errors.WithStack(err)
	}

	iptables, err := s.iptablesRule.List(context.Background())
	if err != nil {
		return errors.WithStack(err)
	}

	chains := core.IptablesRuleList(iptables).ToChains()

	if _, err := s.svr.SetIptablesChains(context.Background(), &pb.IptablesChainsRequest{
		Chains: chains,
	}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (s *Server) recoverTC(uid string, device string) error {
	if err := s.tcRule.DeleteByExperiment(context.Background(), uid); err != nil {
		return errors.WithStack(err)
	}

	tcRules, err := s.tcRule.FindByDevice(context.Background(), device)

	tcs, err := core.TCRuleList(tcRules).ToTCs()
	if err != nil {
		return errors.WithStack(err)
	}

	if _, err := s.svr.SetTcs(context.Background(), &pb.TcsRequest{Tcs: tcs, Device: device}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
