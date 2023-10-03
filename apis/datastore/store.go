//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package datastore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"chichi/apis/datastore/expr"
	"chichi/apis/datastore/warehouses"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/connector/types"

	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/maps"
)

type Store struct {
	ds        *Datastore
	workspace int
	warehouse warehouses.Warehouse
	mu        sync.Mutex // for the events field
	events    [][]any
	closed    atomic.Bool
}

// newStore returns a new Store for the workspace ws.
func newStore(ds *Datastore, ws *state.Workspace) (*Store, error) {
	store := &Store{
		ds:        ds,
		workspace: ws.ID,
	}
	var err error
	store.warehouse, err = openWarehouse(ws.Warehouse.Type, ws.Warehouse.Settings)
	if err != nil {
		return nil, fmt.Errorf("cannot open data warehouse: %s", err)
	}
	go func() {
		ticker := time.NewTicker(flushEventsQueueTimeout)
		for {
			select {
			case <-ticker.C:
				store.mu.Lock()
				events := store.events
				store.events = nil
				store.mu.Unlock()
				if events != nil {
					go store.flushEvents(events)
				}
			}
		}
	}()
	return store, nil
}

// AddEvents adds events to the store.
func (store *Store) AddEvents(events [][]any) {
	store.mustBeOpen()
	store.mu.Lock()
	store.events = append(store.events, events...)
	store.mu.Unlock()
}

// CreateUser creates a user with the given properties and returns its
// identifier.
// isArray indicates which properties have type array.
func (store *Store) CreateUser(ctx context.Context, user map[string]any, isArray map[string]bool) (int, error) {
	store.mustBeOpen()
	b := strings.Builder{}
	b.WriteString("INSERT INTO users (")
	properties := maps.Keys(user)
	for i, name := range properties {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(name)
		b.WriteByte('"')
	}
	b.WriteString(") VALUES (")
	values := make([]any, len(properties))
	for i, name := range properties {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('$')
		b.WriteString(strconv.Itoa(i + 1))
		values[i] = user[name]
	}
	b.WriteString(`) RETURNING "id"`)
	var id int
	err := store.warehouse.QueryRow(ctx, b.String(), values...).Scan(&id)
	if err != nil {
		return 0, err
	}
	err = store.setRedisUserIndex(ctx, IRUser{ID: id, Properties: user}, isArray)
	if err != nil {
		return 0, err
	}
	err = store.ds.redis.Save(ctx).Err()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// DeleteUser deletes the user with identifier id.
// isArray indicates which properties have type array.
//
// If an error occurs with the data warehouse, it returns a DataWarehouseError
// error.
func (store *Store) DeleteUser(ctx context.Context, id int, isArray map[string]bool) error {
	store.mustBeOpen()
	err := store.deleteRedisUserIndex(ctx, id, isArray)
	if err != nil {
		return err
	}
	_, err = store.warehouse.Exec(ctx, `DELETE FROM "users" WHERE "id" = $1`, id)
	return err
}

// DestinationUser returns the external ID of the destination user of the
// action that matches with the corresponding property. If it cannot be
// found, then the empty string and false are returned.
//
// If an error occurs with the data warehouse, it returns a DataWarehouseError
// error.
func (store *Store) DestinationUser(ctx context.Context, action int, property string) (string, bool, error) {
	store.mustBeOpen()
	return store.warehouse.DestinationUser(ctx, action, property)
}

// Events returns the events that satisfy the where condition with only the
// given columns, ordered by order if order is not the zero Property, and in
// range [first,first+limit] with first >= 0 and 0 < limit <= 1000.
//
// If an error occurs with the data warehouse, it returns a DataWarehouseError
// error.
func (store *Store) Events(ctx context.Context, columns []types.Property, where expr.Expr, order types.Property, first, limit int) ([][]any, error) {
	store.mustBeOpen()
	return store.warehouse.Select(ctx, "events", columns, where, order, first, limit)
}

// InitWarehouse initializes the data warehouse creating the events and the
// destinations_users tables.
//
// If an error occurs with the data warehouse, it returns a DataWarehouseError
// error.
func (store *Store) InitWarehouse(ctx context.Context) error {
	store.mustBeOpen()
	return store.warehouse.Init(ctx)
}

// SetDestinationUser sets the destination user relative to the action, with
// the given external user ID and external property.
//
// If an error occurs with the data warehouse, it returns a DataWarehouseError
// error.
func (store *Store) SetDestinationUser(ctx context.Context, connection int, externalUserID, externalProperty string) error {
	store.mustBeOpen()
	return store.warehouse.SetDestinationUser(ctx, connection, externalUserID, externalProperty)
}

// Schemas returns the schemas of users, groups, and events for the relative
// tables. If a table doesn't exist, it won't be included in returned schemas.
//
// If an error occurs with the data warehouse, it returns a DataWarehouseError
// error.
func (store *Store) Schemas(ctx context.Context) (map[string]types.Type, error) {
	store.mustBeOpen()
	tables, err := store.warehouse.Tables(ctx)
	if err != nil {
		return nil, err
	}
	schemas := make(map[string]types.Type)
	for _, table := range tables {
		switch table.Name {
		case "users", "groups", "events":
			properties, err := ColumnsToProperties(table.Columns)
			if err != nil {
				return nil, err
			}
			schemas[table.Name] = types.Object(properties)
		}
	}
	return schemas, nil
}

// UpdateUser updates the properties of the user with identifier id.
// Only the properties in users will be updated.
//
// If an error occurs with the data warehouse, it returns a DataWarehouseError
// error.
func (store *Store) UpdateUser(ctx context.Context, target IRUser, user map[string]any, isArray map[string]bool) error {
	store.mustBeOpen()
	// Since the user contains only the properties to update, merge its property
	// values with the property values of target, then update its index on
	// Redis.
	maps.Copy(target.Properties, user)
	err := store.deleteRedisUserIndex(ctx, target.ID, isArray)
	if err != nil {
		return fmt.Errorf("cannot delete Redis user index: %s", err)
	}
	err = store.setRedisUserIndex(ctx, target, isArray)
	if err != nil {
		return fmt.Errorf("cannot set Redis user index: %s", err)
	}
	// TODO(Gianluca): find a better way to implement persistency.
	// See https://github.com/open2b/chichi/issues/215.
	err = store.ds.redis.Save(ctx).Err()
	if err != nil {
		return fmt.Errorf("cannot save Redis state: %s", err)
	}
	// Update only the properties of user.
	b := &strings.Builder{}
	b.WriteString("UPDATE users SET\n")
	var values []any
	i := 1
	for prop, value := range user {
		if i > 1 {
			b.WriteString(", ")
		}
		b.WriteString(postgres.QuoteIdent(prop))
		b.WriteString(" = $")
		b.WriteString(strconv.Itoa(i))
		values = append(values, value)
		i++
	}
	b.WriteString(`, "timestamp" = now()`)
	b.WriteString("\nWHERE id = $")
	b.WriteString(strconv.Itoa(i))
	values = append(values, target.ID)
	res, err := store.warehouse.Exec(ctx, b.String(), values...)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("BUG: one row should be affected, got %d. Is the Redis index in sync with the content of the users table?", affected)
	}
	return nil
}

// Users returns the users that satisfy the where condition with only the given
// properties, ordered by order if order is not the zero Property, and in range
// [first,first+limit] with first >= 0 and 0 < limit <= 1000.
//
// If an error occurs with the data warehouse, it returns a DataWarehouseError
// error.
func (store *Store) Users(ctx context.Context, properties []types.Property, where expr.Expr, order types.Property, first, limit int) ([]map[string]any, error) {
	store.mustBeOpen()
	columns := PropertiesToColumns(properties)
	rows, err := store.warehouse.Select(ctx, "users", columns, where, order, first, limit)
	if err != nil {
		return nil, err
	}
	users := make([]map[string]any, len(rows))
	for i, row := range rows {
		users[i], _ = deserializeRowAsMap(properties, row)
	}
	return users, nil
}

// UsersSlice is like Users but returns the users as a slice.
//
// If an error occurs with the data warehouse, it returns a DataWarehouseError
// error.
func (store *Store) UsersSlice(ctx context.Context, properties []types.Property, where expr.Expr, order types.Property, first, limit int) ([][]any, error) {
	store.mustBeOpen()
	columns := PropertiesToColumns(properties)
	rows, err := store.warehouse.Select(ctx, "users", columns, where, order, first, limit)
	if err != nil {
		return nil, err
	}
	users := make([][]any, len(rows))
	for i, row := range rows {
		users[i] = deserializeRowAsSlice(properties, row)
	}
	return users, nil
}

// UsersWithCommonValue returns the users with at least one common property
// value with user. isArray indicates which properties have type array.
//
// If an error occurs with the data warehouse, it returns a DataWarehouseError
// error.
func (store *Store) UsersWithCommonValue(ctx context.Context, user map[string]any, isArray map[string]bool) ([]IRUser, error) {

	store.mustBeOpen()

	// Determine the property-value pairs to check on Redis.
	propertyKeys := []string{}
	for p, v := range user {
		if !zero(v, isArray[p]) {
			key := redisPropertyKey(p, v)
			propertyKeys = append(propertyKeys, key)
		}
	}

	// TODO(Gianluca): remove this panic and handle the situation properly.
	// See the issue https://github.com/open2b/chichi/issues/253.
	if len(propertyKeys) == 0 {
		panic("BUG: see the issue https://github.com/open2b/chichi/issues/253")
	}

	// Retrieve property-value pairs from Redis and collect the GIDs.
	vals, err := store.ds.redis.MGet(ctx, propertyKeys...).Result()
	if err != nil {
		return nil, err
	}
	gids := map[int]struct{}{}
	for _, v := range vals {
		if v == nil { // no matches for this property-value pair.
			continue
		}
		ids, err := deserializeIDs(v.(string))
		if err != nil {
			return nil, err
		}
		for _, gid := range ids {
			gids[gid] = struct{}{}
		}
	}
	if len(gids) == 0 {
		return []IRUser{}, nil
	}

	// Retrieve the properties for every user.
	userKeys := make([]string, len(gids))
	i := 0
	for gid := range gids {
		userKeys[i] = redisUserKey(gid)
		i++
	}
	slices.Sort(userKeys)
	rawUsers, err := store.ds.redis.MGet(ctx, userKeys...).Result()
	if err != nil {
		return nil, err
	}
	users := make([]IRUser, len(rawUsers))
	for i, user := range rawUsers {
		u := IRUser{}
		u.Properties = redisJSONDeserialize(user.(string)).(map[string]any)
		u.ID = int(u.Properties["id"].(float64))
		delete(u.Properties, "id")
		users[i] = u
	}

	return users, nil
}

// close closes the store.
// It flushes the events and closes the Redis database and the data warehouse.
// It panics if it has already been called.
func (store *Store) close() error {
	if store.closed.Swap(true) {
		panic("apis/datastore/store already closed")
	}
	store.mu.Lock()
	if len(store.events) > 0 {
		store.flushEvents(store.events)
		store.events = nil
	}
	store.mu.Unlock()
	err := store.ds.redis.Close()
	if err != nil {
		err = fmt.Errorf("error occurred closing Redis database: %s", err)
	}
	err2 := store.warehouse.Close()
	if err2 != nil {
		err2 = fmt.Errorf("error occurred closing data warehouse: %s", err)
		if err != nil {
			err = errors.New(err.Error() + "\n\tand also " + err2.Error())
		}
	}
	return err
}

// deleteRedisUserIndex deletes from the index the user with the given GID.
// isArray indicates which properties have type array.
func (store *Store) deleteRedisUserIndex(ctx context.Context, id int, isArray map[string]bool) error {

	// Retrieve the user.
	rawUser, err := store.ds.redis.Get(ctx, redisUserKey(id)).Result()
	if err != nil {
		if err == redis.Nil {
			err = nil
		}
		return err
	}
	user := redisJSONDeserialize(rawUser).(map[string]any)
	delete(user, "id")

	// Remove the user ID from the "property:" keys.
	for p, v := range user {
		var values []any
		if isArray[p] {
			// v is an array value.
			rv := reflect.ValueOf(v)
			if rv.Len() == 0 {
				continue
			}
			values = make([]any, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				values[i] = rv.Index(i).Interface()
			}
		} else {
			// v is a scalar value.
			if zero(v, isArray[p]) {
				continue
			}
			values = []any{v}
		}
		for _, v := range values {
			key := redisPropertyKey(p, v)
			rawIDs, err := store.ds.redis.Get(ctx, key).Result()
			if err != nil {
				return err
			}
			ids, err := deserializeIDs(rawIDs)
			if err != nil {
				return err
			}
			ids = slices.DeleteFunc(ids, func(id2 int) bool {
				return id2 == id
			})
			if len(ids) > 0 {
				err = store.ds.redis.Set(ctx, key, serializeIDs(ids), 0).Err()
				if err != nil {
					return err
				}
			} else {
				err = store.ds.redis.Del(ctx, key).Err()
				if err != nil {
					return err
				}
			}
		}
	}

	// Remove the "user:<id>" key.
	err = store.ds.redis.Del(ctx, redisUserKey(id)).Err()

	return err
}

// mustBeOpen panics if store has been closed.
func (store *Store) mustBeOpen() {
	if store.closed.Load() {
		panic("apis/datastore/store is closed")
	}
}

func (store *Store) setRedisUserIndex(ctx context.Context, user IRUser, isArray map[string]bool) error {

	// TODO(Gianluca): should we keep on Redis only the identifiers? See the
	// issue: https://github.com/open2b/chichi/issues/243

	// Write the "property:" keys.
	for p, v := range user.Properties {
		var values []any
		if isArray[p] {
			// v is an array value.
			rv := reflect.ValueOf(v)
			if rv.Len() == 0 {
				continue
			}
			values = make([]any, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				values[i] = rv.Index(i).Interface()
			}
		} else {
			// v is a scalar value.
			if zero(v, isArray[p]) {
				continue
			}
			values = []any{v}
		}
		for _, v := range values {
			key := redisPropertyKey(p, v)
			current, err := store.ds.redis.Get(ctx, key).Result()
			if err != nil && err != redis.Nil {
				return fmt.Errorf("cannot GET value from Redis: %s", err)
			}
			ids, err := deserializeIDs(current)
			if err != nil {
				return err
			}
			if !slices.Contains(ids, user.ID) {
				ids = append(ids, user.ID)
			}
			newVal := serializeIDs(ids)
			err = store.ds.redis.Set(ctx, key, newVal, 0).Err()
			if err != nil {
				return fmt.Errorf("cannot SET value on Redis: %s", err)
			}
		}
	}

	// Write the "user:" key.
	userToSerialize := map[string]any{}
	maps.Copy(userToSerialize, user.Properties)
	userToSerialize["id"] = user.ID
	err := store.ds.redis.Set(ctx, redisUserKey(user.ID), redisJSONSerialize(userToSerialize), 0).Err()

	return err
}

// IRUser holds the information of a user necessary for the identity resolution
// process.
type IRUser struct {
	ID         int
	Properties map[string]any
}

func deserializeIDs(s string) ([]int, error) {
	if s == "" {
		return []int{}, nil
	}
	ids := []int{}
	rawGids := strings.Split(s, ",")
	for _, r := range rawGids {
		gid, err := strconv.Atoi(r)
		if err != nil {
			return nil, errors.New("invalid IDs")
		}
		ids = append(ids, gid)
	}
	return ids, nil
}

func redisJSONDeserialize(raw string) any {
	// TODO: improve or remove this function.
	var v any
	err := json.Unmarshal([]byte(raw), &v)
	if err != nil {
		log.Panic(err)
	}
	return v
}

func redisJSONSerialize(v any) string {
	// TODO: improve or remove this function.
	data, err := json.Marshal(v)
	if err != nil {
		log.Panic(err)
	}
	return string(data)
}

// redisPropertyKey returns a Redis key in the form:
//
//	property:<property>:<value>
func redisPropertyKey(property string, value any) string {
	b := strings.Builder{}
	b.WriteString("property:")
	b.WriteString(property)
	b.WriteByte(':')
	b.WriteString(redisJSONSerialize(value))
	return b.String()
}

// redisPropertyKey returns a Redis key in the form:
//
//	user:<id>
func redisUserKey(id int) string {
	return "user:" + strconv.Itoa(id)
}

func serializeIDs(ids []int) string {
	b := strings.Builder{}
	for i, id := range ids {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Itoa(id))
	}
	return b.String()
}

func zero(v any, isArray bool) bool {
	// Keep in sync with the function 'zero' in 'apis/userswarehouse'.
	if v == nil {
		return true
	}
	if isArray {
		return len(v.([]any)) == 0
	}
	return v == "" || v == 0
}
