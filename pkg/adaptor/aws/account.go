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

package aws

import (
	"errors"
	"fmt"
	"strings"

	"hcm/pkg/adaptor/types"
	"hcm/pkg/criteria/errf"
	"hcm/pkg/kit"
	"hcm/pkg/logs"

	"github.com/aws/aws-sdk-go/service/sts"
)

var _ types.AccountInterface = new(amazon)

func validateAccountCheckOption(opt *types.AccountCheckOption) error {
	if opt == nil {
		return errf.New(errf.InvalidParameter, "account check option is required")
	}

	if opt.Aws == nil {
		return errf.New(errf.InvalidParameter, "aws account info is required")
	}

	if len(opt.Aws.AccountCid) == 0 {
		return errf.New(errf.InvalidParameter, "account cid is required")
	}

	if len(opt.Aws.IamUserName) == 0 {
		return errf.New(errf.InvalidParameter, "iam user name is required")
	}

	return nil
}

// AccountCheck check account authentication information(account id and iam user name) and permissions.
// GetCallerIdentity: https://docs.aws.amazon.com/STS/latest/APIReference/API_GetCallerIdentity.html
func (am *amazon) AccountCheck(kt *kit.Kit, secret *types.Secret, opt *types.AccountCheckOption) error {
	if err := validateSecret(secret); err != nil {
		return err
	}

	if err := validateAccountCheckOption(opt); err != nil {
		return err
	}

	client, err := am.stsClient(secret.Aws)
	if err != nil {
		return fmt.Errorf("init aws client failed, err: %v", err)
	}

	req := new(sts.GetCallerIdentityInput)
	resp, err := client.GetCallerIdentityWithContext(kt.Ctx, req)
	if err != nil {
		logs.Errorf("describe regions failed, err: %v, rid: %s", err, kt.Rid)
		return err
	}

	if resp.Account == nil {
		return errors.New("get caller identity return account is nil")
	}

	// check account info: account id、user name
	if *resp.Account != opt.Aws.AccountCid {
		return fmt.Errorf("account id does not match the account to which the secret belongs")
	}

	if resp.Arn == nil {
		return errors.New("get caller identity return arn is nil")
	}

	split := strings.Split(*resp.Arn, "/")
	if split[len(split)-1] != opt.Aws.IamUserName {
		return fmt.Errorf("iam user name does not match the account to which the secret belongs")
	}

	return nil
}
