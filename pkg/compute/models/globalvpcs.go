// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SGlobalVpcManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
}

var GlobalVpcManager *SGlobalVpcManager

func init() {
	GlobalVpcManager = &SGlobalVpcManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SGlobalVpc{},
			"globalvpcs_tbl",
			"globalvpc",
			"globalvpcs",
		),
	}
	GlobalVpcManager.SetVirtualObject(GlobalVpcManager)
}

type SGlobalVpc struct {
	db.SEnabledStatusStandaloneResourceBase
}

func (manager *SGlobalVpcManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, manager)
}

func (self *SGlobalVpc) ValidateDeleteCondition(ctx context.Context) error {
	vpcs, err := self.GetVpcs()
	if err != nil {
		return errors.Wrap(err, "self.GetVpcs")
	}
	if len(vpcs) > 0 {
		return fmt.Errorf("not an empty globalvpc")
	}
	return self.SEnabledStatusStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SGlobalVpc) GetVpcs() ([]SVpc, error) {
	vpcs := []SVpc{}
	q := VpcManager.Query().Equals("globalvpc_id", self.Id)
	err := db.FetchModelObjects(VpcManager, q, &vpcs)
	if err != nil {
		return nil, err
	}
	return vpcs, nil
}

func (self *SGlobalVpc) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.GlobalVpcDetails, error) {
	return api.GlobalVpcDetails{}, nil
}

func (manager *SGlobalVpcManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.GlobalVpcDetails {
	rows := make([]api.GlobalVpcDetails, len(objs))
	stdRows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.GlobalVpcDetails{
			EnabledStatusStandaloneResourceDetails: stdRows[i],
		}
	}
	return rows
}

func (manager *SGlobalVpcManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.GlobalVpcCreateInput) (api.GlobalVpcCreateInput, error) {
	input.Status = api.GLOBAL_VPC_STATUS_AVAILABLE
	var err error
	input.EnabledStatusStandaloneResourceCreateInput, err = manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusStandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData")
	}
	return input, nil
}

func (self *SGlobalVpc) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return self.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

// 全局VPC列表
func (manager *SGlobalVpcManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GlobalVpcListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SGlobalVpcManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GlobalVpcListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SGlobalVpcManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (self *SGlobalVpc) ValidateUpdateCondition(ctx context.Context) error {
	return self.SEnabledStatusStandaloneResourceBase.ValidateUpdateCondition(ctx)
}
