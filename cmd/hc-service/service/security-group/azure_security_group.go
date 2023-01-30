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

package securitygroup

import (
	"hcm/pkg/adaptor/types"
	corecloud "hcm/pkg/api/core/cloud"
	protocloud "hcm/pkg/api/data-service/cloud"
	proto "hcm/pkg/api/hc-service"
	"hcm/pkg/criteria/errf"
	"hcm/pkg/dal/dao/tools"
	"hcm/pkg/logs"
	"hcm/pkg/rest"
)

// CreateAzureSecurityGroup create azure security group.
func (g *securityGroup) CreateAzureSecurityGroup(cts *rest.Contexts) (interface{}, error) {
	req := new(proto.AzureSecurityGroupCreateReq)
	if err := cts.DecodeInto(req); err != nil {
		return nil, errf.NewFromErr(errf.DecodeRequestFailed, err)
	}

	if err := req.Validate(); err != nil {
		return nil, errf.NewFromErr(errf.InvalidParameter, err)
	}

	client, err := g.ad.Azure(cts.Kit, req.AccountID)
	if err != nil {
		return nil, err
	}

	opt := &types.AzureSecurityGroupOption{
		ResourceGroupName: req.ResourceGroupName,
		Region:            req.Region,
		Name:              req.Name,
	}
	sg, err := client.CreateSecurityGroup(cts.Kit, opt)
	if err != nil {
		logs.Errorf("request adaptor to create azure security group failed, err: %v, opt: %v, rid: %s", err, opt,
			cts.Kit.Rid)
		return nil, err
	}

	createReq := &protocloud.SecurityGroupBatchCreateReq[corecloud.AzureSecurityGroupExtension]{
		SecurityGroups: []protocloud.SecurityGroupBatchCreate[corecloud.AzureSecurityGroupExtension]{
			{
				CloudID:   *sg.ID,
				BkBizID:   req.BkBizID,
				Region:    req.Region,
				Name:      *sg.Name,
				Memo:      req.Memo,
				AccountID: req.AccountID,
				Extension: &corecloud.AzureSecurityGroupExtension{
					Etag:              sg.Etag,
					FlushConnection:   sg.Properties.FlushConnection,
					ResourceGUID:      sg.Properties.ResourceGUID,
					ProvisioningState: string(*sg.Properties.ProvisioningState),
				},
			},
		},
	}
	result, err := g.dataCli.Azure.SecurityGroup.BatchCreateSecurityGroup(cts.Kit.Ctx, cts.Kit.Header(), createReq)
	if err != nil {
		logs.Errorf("request dataservice to BatchCreateSecurityGroup failed, err: %v, rid: %s", err, cts.Kit.Rid)
		return nil, err
	}

	return result, nil
}

// DeleteAzureSecurityGroup delete azure security group.
func (g *securityGroup) DeleteAzureSecurityGroup(cts *rest.Contexts) (interface{}, error) {
	id := cts.PathParameter("id").String()
	if len(id) == 0 {
		return nil, errf.New(errf.InvalidParameter, "id is required")
	}

	sg, err := g.dataCli.Azure.SecurityGroup.GetSecurityGroup(cts.Kit.Ctx, cts.Kit.Header(), id)
	if err != nil {
		logs.Errorf("request dataservice get azure security group failed, err: %v, id: %s, rid: %s", err, id,
			cts.Kit.Rid)
		return nil, err
	}

	client, err := g.ad.Azure(cts.Kit, sg.AccountID)
	if err != nil {
		return nil, err
	}

	opt := &types.AzureSecurityGroupOption{
		ResourceGroupName: sg.Extension.ResourceGroupName,
		Region:            sg.Region,
		Name:              sg.Name,
	}
	if err := client.DeleteSecurityGroup(cts.Kit, opt); err != nil {
		logs.Errorf("request adaptor to delete azure security group failed, err: %v, opt: %v, rid: %s", err, opt,
			cts.Kit.Rid)
		return nil, err
	}

	req := &protocloud.SecurityGroupBatchDeleteReq{
		Filter: tools.EqualExpression("id", id),
	}
	if err := g.dataCli.Global.SecurityGroup.BatchDeleteSecurityGroup(cts.Kit.Ctx, cts.Kit.Header(), req); err != nil {
		logs.Errorf("request dataservice delete azure security group failed, err: %v, id: %s, rid: %s", err, id,
			cts.Kit.Rid)
		return nil, err
	}

	return nil, nil
}

// UpdateAzureSecurityGroup update azure security group.
func (g *securityGroup) UpdateAzureSecurityGroup(cts *rest.Contexts) (interface{}, error) {
	id := cts.PathParameter("id").String()
	if len(id) == 0 {
		return nil, errf.New(errf.InvalidParameter, "id is required")
	}

	req := new(proto.SecurityGroupUpdateReq)
	if err := cts.DecodeInto(req); err != nil {
		return nil, errf.NewFromErr(errf.DecodeRequestFailed, err)
	}

	if err := req.Validate(); err != nil {
		return nil, errf.NewFromErr(errf.InvalidParameter, err)
	}

	// 云上仅支持更新Name。
	if len(req.Name) != 0 {
		sg, err := g.dataCli.Azure.SecurityGroup.GetSecurityGroup(cts.Kit.Ctx, cts.Kit.Header(), id)
		if err != nil {
			logs.Errorf("request dataservice get azure security group failed, err: %v, id: %s, rid: %s", err, id,
				cts.Kit.Rid)
			return nil, err
		}

		client, err := g.ad.Azure(cts.Kit, sg.AccountID)
		if err != nil {
			return nil, err
		}

		opt := &types.AzureSecurityGroupOption{
			Region:            sg.Region,
			Name:              req.Name,
			ResourceGroupName: sg.Extension.ResourceGroupName,
		}
		if err := client.UpdateSecurityGroup(cts.Kit, opt); err != nil {
			logs.Errorf("request adaptor to UpdateSecurityGroup failed, err: %v, opt: %v, rid: %s", err, opt,
				cts.Kit.Rid)
			return nil, err
		}
	}

	updateReq := &protocloud.SecurityGroupBatchUpdateReq[corecloud.AzureSecurityGroupExtension]{
		SecurityGroups: []protocloud.SecurityGroupBatchUpdate[corecloud.AzureSecurityGroupExtension]{
			{
				ID:   id,
				Name: req.Name,
				Memo: req.Memo,
			},
		},
	}
	if err := g.dataCli.Azure.SecurityGroup.BatchUpdateSecurityGroup(cts.Kit.Ctx, cts.Kit.Header(),
		updateReq); err != nil {

		logs.Errorf("request dataservice BatchUpdateSecurityGroup failed, err: %v, id: %s, rid: %s", err, id,
			cts.Kit.Rid)
		return nil, err
	}

	return nil, nil
}

// SyncAzureSecurityGroup create azure security group.
func (g *securityGroup) SyncAzureSecurityGroup(cts *rest.Contexts) (interface{}, error) {

	req, err := g.decodeSecurityGroupSyncReq(cts)
	if err != nil {
		return nil, err
	}

	yunMap, err := g.getDatasFromAzureForSecurityGroupSync(cts, req)
	if err != nil {
		return nil, err
	}

	dsMap, err := g.getDatasFromDSForSecurityGroupSync(cts, req)
	if err != nil {
		return nil, err
	}

	err = g.diffAzureSecurityGroupSync(cts, yunMap, dsMap, req)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// getDatasFromAzureForSecurityGroupSync get datas from cloud
func (g *securityGroup) getDatasFromAzureForSecurityGroupSync(cts *rest.Contexts,
	req *proto.SecurityGroupSyncReq) (map[string]*proto.SecurityGroupSyncAzureDiff, error) {

	client, err := g.ad.Azure(cts.Kit, req.AccountID)
	if err != nil {
		return nil, err
	}

	listOpt := &types.AzureSecurityGroupListOption{
		ResourceGroupName: req.ResourceGroupName,
	}
	result, err := client.ListSecurityGroup(cts.Kit, listOpt)
	if err != nil {
		logs.Errorf("request adaptor to list azure security group failed, err: %v, rid: %s", err, cts.Kit.Rid)
		return nil, err
	}

	yunMap := make(map[string]*proto.SecurityGroupSyncAzureDiff)
	for _, one := range result {
		sg := new(proto.SecurityGroupSyncAzureDiff)
		sg.IsMarked = false
		sg.SecurityGroup = one
		yunMap[*one.ID] = sg
	}

	return yunMap, nil
}

// diffAzureSecurityGroupSync diff cloud data-service
func (g *securityGroup) diffAzureSecurityGroupSync(cts *rest.Contexts, yunMap map[string]*proto.SecurityGroupSyncAzureDiff,
	dsMap map[string]*proto.SecurityGroupSyncDS, req *proto.SecurityGroupSyncReq) error {

	addCloudIDs := getAddCloudIDs(yunMap, dsMap)
	deleteCloudIDs, updateCloudIDs := getDeleteAndUpdateCloudIDs(dsMap)

	if len(deleteCloudIDs) > 0 {
		err := g.diffSecurityGroupSyncDelete(cts, deleteCloudIDs)
		if err != nil {
			return err
		}
		err = g.diffAzureSGRuleSyncDelete(cts, deleteCloudIDs, dsMap)
		if err != nil {
			return err
		}
	}
	if len(updateCloudIDs) > 0 {
		err := g.diffAzureSecurityGroupSyncUpdate(cts, yunMap, dsMap, updateCloudIDs)
		if err != nil {
			return err
		}
		err = g.diffAzureSGRuleSyncUpdate(cts, updateCloudIDs, req, dsMap)
		if err != nil {
			return err
		}
	}
	if len(addCloudIDs) > 0 {
		ids, err := g.diffAzureSecurityGroupSyncAdd(cts, yunMap, req, addCloudIDs)
		if err != nil {
			return err
		}
		err = g.diffAzureSGRuleSyncAdd(cts, ids, req)
		if err != nil {
			return err
		}
	}

	return nil
}

// diffAzuerSecurityGroupSyncAdd for add
func (g *securityGroup) diffAzureSecurityGroupSyncAdd(cts *rest.Contexts, yunMap map[string]*proto.SecurityGroupSyncAzureDiff,
	req *proto.SecurityGroupSyncReq, addCloudIDs []string) ([]string, error) {

	createReq := &protocloud.SecurityGroupBatchCreateReq[corecloud.AzureSecurityGroupExtension]{
		SecurityGroups: []protocloud.SecurityGroupBatchCreate[corecloud.AzureSecurityGroupExtension]{},
	}

	for _, id := range addCloudIDs {
		securityGroup := protocloud.SecurityGroupBatchCreate[corecloud.AzureSecurityGroupExtension]{
			CloudID:   *yunMap[id].SecurityGroup.ID,
			BkBizID:   -1,
			Region:    *yunMap[id].SecurityGroup.Location,
			Name:      *yunMap[id].SecurityGroup.Name,
			Memo:      nil,
			AccountID: req.AccountID,
			Extension: &corecloud.AzureSecurityGroupExtension{
				Etag:              yunMap[id].SecurityGroup.Etag,
				FlushConnection:   yunMap[id].SecurityGroup.Properties.FlushConnection,
				ResourceGUID:      yunMap[id].SecurityGroup.Properties.ResourceGUID,
				ProvisioningState: string(*yunMap[id].SecurityGroup.Properties.ProvisioningState),
			},
		}
		createReq.SecurityGroups = append(createReq.SecurityGroups, securityGroup)
	}
	results, err := g.dataCli.Azure.SecurityGroup.BatchCreateSecurityGroup(cts.Kit.Ctx, cts.Kit.Header(), createReq)
	if err != nil {
		logs.Errorf("request dataservice to BatchCreateSecurityGroup failed, err: %v, rid: %s", err, cts.Kit.Rid)
		return nil, err
	}

	return results.IDs, nil
}

// diffAzureSecurityGroupSyncUpdate for update
func (g *securityGroup) diffAzureSecurityGroupSyncUpdate(cts *rest.Contexts, yunMap map[string]*proto.SecurityGroupSyncAzureDiff,
	dsMap map[string]*proto.SecurityGroupSyncDS, updateCloudIDs []string) error {

	updateReq := &protocloud.SecurityGroupBatchUpdateReq[corecloud.AzureSecurityGroupExtension]{
		SecurityGroups: []protocloud.SecurityGroupBatchUpdate[corecloud.AzureSecurityGroupExtension]{},
	}

	for _, id := range updateCloudIDs {
		if *yunMap[id].SecurityGroup.Name == dsMap[id].HcSecurityGroup.Name {
			continue
		}
		securityGroup := protocloud.SecurityGroupBatchUpdate[corecloud.AzureSecurityGroupExtension]{
			ID:   dsMap[id].HcSecurityGroup.ID,
			Name: *yunMap[id].SecurityGroup.Name,
		}
		updateReq.SecurityGroups = append(updateReq.SecurityGroups, securityGroup)
	}
	if err := g.dataCli.Azure.SecurityGroup.BatchUpdateSecurityGroup(cts.Kit.Ctx, cts.Kit.Header(),
		updateReq); err != nil {
		logs.Errorf("request dataservice BatchUpdateSecurityGroup failed, err: %v, rid: %s", err, cts.Kit.Rid)
		return err
	}

	return nil
}
