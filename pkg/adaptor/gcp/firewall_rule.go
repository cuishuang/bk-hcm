/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 混合云管理平台 (BlueKing - Hybrid Cloud Management System) available.
 * Copyright (C) 2022 THL A29 Limited,
 * a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on
 * an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 *
 * to the current version of the project delivered to anyone in the future.
 */

package gcp

import (
	"hcm/pkg/adaptor/types/core"
	"hcm/pkg/adaptor/types/firewall-rule"
	"hcm/pkg/criteria/errf"
	"hcm/pkg/kit"
	"hcm/pkg/logs"

	"google.golang.org/api/compute/v1"
)

// ListFirewallRule list firewall rule.
// reference: https://cloud.google.com/compute/docs/reference/rest/v1/firewalls/list
func (g *Gcp) ListFirewallRule(kt *kit.Kit, opt *firewallrule.ListOption) (*compute.FirewallList, error) {
	if opt == nil {
		return nil, errf.New(errf.InvalidParameter, "list option is required")
	}

	if err := opt.Validate(); err != nil {
		return nil, errf.NewFromErr(errf.InvalidParameter, err)
	}

	client, err := g.clientSet.computeClient(kt)
	if err != nil {
		return nil, err
	}

	request := client.Firewalls.List(g.CloudProjectID()).Context(kt.Ctx)

	var call *compute.FirewallsListCall
	if opt.Page != nil {
		call = request.MaxResults(opt.Page.PageSize).PageToken(opt.Page.PageToken)
	} else {
		call = request.MaxResults(core.GcpQueryLimit)
	}

	resp, err := call.Do()
	if err != nil {
		logs.Errorf("list firewall rule failed, err: %v, opt: %v, rid: %s", err, opt, kt.Rid)
		return nil, err
	}

	return resp, nil
}

// UpdateFirewallRule update firewall rule.
// reference: https://cloud.google.com/compute/docs/reference/rest/v1/firewalls/patch
func (g *Gcp) UpdateFirewallRule(kt *kit.Kit, opt *firewallrule.UpdateOption) error {
	if opt == nil {
		return errf.New(errf.InvalidParameter, "update option is required")
	}

	if err := opt.Validate(); err != nil {
		return errf.NewFromErr(errf.InvalidParameter, err)
	}

	client, err := g.clientSet.computeClient(kt)
	if err != nil {
		return err
	}

	update := &compute.Firewall{
		Description:           opt.GcpFirewallRule.Description,
		DestinationRanges:     opt.GcpFirewallRule.DestinationRanges,
		Disabled:              opt.GcpFirewallRule.Disabled,
		Priority:              opt.GcpFirewallRule.Priority,
		SourceRanges:          opt.GcpFirewallRule.SourceRanges,
		SourceTags:            opt.GcpFirewallRule.SourceTags,
		TargetTags:            opt.GcpFirewallRule.TargetTags,
		SourceServiceAccounts: opt.GcpFirewallRule.SourceServiceAccounts,
		TargetServiceAccounts: opt.GcpFirewallRule.TargetServiceAccounts,
	}

	if len(opt.GcpFirewallRule.Allowed) != 0 {
		update.Allowed = make([]*compute.FirewallAllowed, 0, len(opt.GcpFirewallRule.Allowed))
		for _, one := range opt.GcpFirewallRule.Allowed {
			update.Allowed = append(update.Allowed, &compute.FirewallAllowed{
				IPProtocol: one.Protocol,
				Ports:      one.Port,
			})
		}
	}

	if len(opt.GcpFirewallRule.Denied) != 0 {
		update.Denied = make([]*compute.FirewallDenied, 0, len(opt.GcpFirewallRule.Denied))
		for _, one := range opt.GcpFirewallRule.Allowed {
			update.Denied = append(update.Denied, &compute.FirewallDenied{
				IPProtocol: one.Protocol,
				Ports:      one.Port,
			})
		}
	}

	_, err = client.Firewalls.Patch(g.CloudProjectID(), opt.CloudID, update).Do()
	if err != nil {
		logs.Errorf("patch firewall rule failed, err: %v, id: %s, update: %v, rid: %s", err, opt.CloudID,
			update, kt.Rid)
	}

	return nil
}

// DeleteFirewallRule delete firewall rule.
// reference: https://cloud.google.com/compute/docs/reference/rest/v1/firewalls/delete
func (g *Gcp) DeleteFirewallRule(kt *kit.Kit, opt *firewallrule.DeleteOption) error {
	if opt == nil {
		return errf.New(errf.InvalidParameter, "delete option is required")
	}

	if err := opt.Validate(); err != nil {
		return errf.NewFromErr(errf.InvalidParameter, err)
	}

	client, err := g.clientSet.computeClient(kt)
	if err != nil {
		return err
	}

	_, err = client.Firewalls.Delete(g.CloudProjectID(), opt.CloudID).Do()
	if err != nil {
		logs.Errorf("delete firewall rule failed, err: %v, id: %s, rid: %s", err, opt.CloudID, kt.Rid)
	}

	return nil
}
