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
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SStorageResourceBase struct {
	StorageId string `width:"36" charset:"ascii" nullable:"true" list:"user" index:"true" create:"optional"`
}

type SStorageResourceBaseManager struct {
	SZoneResourceBaseManager
	SManagedResourceBaseManager
}

func (self *SStorageResourceBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) api.StorageResourceInfo {
	return api.StorageResourceInfo{}
}

func (manager *SStorageResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.StorageResourceInfo {
	rows := make([]api.StorageResourceInfo, len(objs))
	storageIds := make([]string, len(objs))
	for i := range objs {
		var base *SStorageResourceBase
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil {
			storageIds[i] = base.StorageId
		}
	}

	storages := make(map[string]SStorage)
	err := db.FetchStandaloneObjectsByIds(StorageManager, storageIds, storages)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return nil
	}

	zoneList := make([]interface{}, len(rows))
	managerList := make([]interface{}, len(rows))

	for i := range rows {
		rows[i] = api.StorageResourceInfo{}
		if _, ok := storages[storageIds[i]]; ok {
			storage := storages[storageIds[i]]
			rows[i].Storage = storage.Name
			rows[i].StorageStatus = storage.Status
			rows[i].StorageType = storage.StorageType
			rows[i].MediumType = storage.MediumType
			rows[i].ManagerId = storage.ManagerId
			rows[i].ZoneId = storage.ZoneId
		}
		zoneList[i] = &SZoneResourceBase{rows[i].ZoneId}
		managerList[i] = &SManagedResourceBase{rows[i].ManagerId}
	}

	zoneRows := manager.SZoneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, zoneList, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, managerList, fields, isList)
	for i := range rows {
		rows[i].ZoneResourceInfo = zoneRows[i]
		rows[i].ManagedResourceInfo = managerRows[i]
	}

	return rows
}

func (manager *SStorageResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.StorageFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.Storage) > 0 {
		storageObj, err := StorageManager.FetchByIdOrName(userCred, query.Storage)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(StorageManager.Keyword(), query.Storage)
			} else {
				return nil, errors.Wrap(err, "StorageManager.FetchByIdOrName")
			}
		}
		q = q.Equals("storage_id", storageObj.GetId())
	}
	subq := StorageManager.Query("id").Snapshot()
	subq, err := manager.SZoneResourceBaseManager.ListItemFilter(ctx, subq, userCred, query.ZonalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemFilter")
	}
	subq, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, subq, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	if query.Share != nil && *query.Share {
		subq = subq.Filter(sqlchemy.NotIn(subq.Field("storage_type"), api.STORAGE_LOCAL_TYPES))
	}
	if query.Local != nil && *query.Local {
		subq = subq.Filter(sqlchemy.In(subq.Field("storage_type"), api.STORAGE_LOCAL_TYPES))
	}
	if subq.IsAltered() {
		q = q.Filter(sqlchemy.In(q.Field("storage_id"), subq.SubQuery()))
	}
	return q, nil
}

func (manager *SStorageResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "storage":
		storages := StorageManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(storages.Field("name", field))
		q = q.Join(storages, sqlchemy.Equals(q.Field("storage_id"), storages.Field("id")))
		q.GroupBy(storages.Field("name"))
		return q, nil
	case "storage_type", "medium_type":
		storages := StorageManager.Query(field, "id").Distinct().SubQuery()
		q.AppendField(storages.Field(field))
		q = q.Join(storages, sqlchemy.Equals(q.Field("storage_id"), storages.Field("id")))
		q.GroupBy(storages.Field(field))
		return q, nil
	case "manager", "account", "provider", "brand":
		storages := StorageManager.Query("id", "manager_id").SubQuery()
		q = q.LeftJoin(storages, sqlchemy.Equals(q.Field("storage_id"), storages.Field("id")))
		return manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	default:
		storages := StorageManager.Query("id", "zone_id").SubQuery()
		q = q.LeftJoin(storages, sqlchemy.Equals(q.Field("storage_id"), storages.Field("id")))
		q, err := manager.SZoneResourceBaseManager.QueryDistinctExtraField(q, field)
		if err == nil {
			return q, nil
		}
	}
	return q, httperrors.ErrNotFound
}

func (manager *SStorageResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.StorageFilterListInput,
) (*sqlchemy.SQuery, error) {
	q, orders, fields := manager.GetOrderBySubQuery(q, userCred, query)
	if len(orders) > 0 {
		q = db.OrderByFields(q, orders, fields)
	}
	return q, nil
}

func (manager *SStorageResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.StorageFilterListInput,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	storageQ := StorageManager.Query("id", "name")
	var orders []string
	var fields []sqlchemy.IQueryField

	if db.NeedOrderQuery(manager.SZoneResourceBaseManager.GetOrderByFields(query.ZonalFilterListInput)) {
		var zoneOrders []string
		var zoneFields []sqlchemy.IQueryField
		storageQ, zoneOrders, zoneFields = manager.SZoneResourceBaseManager.GetOrderBySubQuery(storageQ, userCred, query.ZonalFilterListInput)
		if len(zoneOrders) > 0 {
			orders = append(orders, zoneOrders...)
			fields = append(fields, zoneFields...)
		}
	}
	if db.NeedOrderQuery(manager.SManagedResourceBaseManager.GetOrderByFields(query.ManagedResourceListInput)) {
		var manOrders []string
		var manFields []sqlchemy.IQueryField
		storageQ, manOrders, manFields = manager.SManagedResourceBaseManager.GetOrderBySubQuery(storageQ, userCred, query.ManagedResourceListInput)
		if len(manOrders) > 0 {
			orders = append(orders, manOrders...)
			fields = append(fields, manFields...)
		}
	}
	if db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		subq := storageQ.SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("storage_id"), subq.Field("id")))
		if db.NeedOrderQuery([]string{query.OrderByStorage}) {
			orders = append(orders, query.OrderByStorage)
			fields = append(fields, subq.Field("name"))
		}
	}
	return q, orders, fields
}

func (manager *SStorageResourceBaseManager) GetOrderByFields(query api.StorageFilterListInput) []string {
	fields := make([]string, 0)
	zoneFields := manager.SZoneResourceBaseManager.GetOrderByFields(query.ZonalFilterListInput)
	fields = append(fields, zoneFields...)
	managerFields := manager.SManagedResourceBaseManager.GetOrderByFields(query.ManagedResourceListInput)
	fields = append(fields, managerFields...)
	fields = append(fields, query.OrderByStorage)
	return fields
}
