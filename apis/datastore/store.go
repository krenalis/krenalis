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
	"strconv"
	"strings"
	"sync"
	"time"

	"chichi/apis/datastore/warehouses"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/connector/types"

	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/maps"
)

type (
	Expr  = warehouses.Expr
	Row   = warehouses.Row
	Where = warehouses.Where
	Error = warehouses.Error
)

type Store struct {
	workspace int
	redis     *redis.Client
	warehouse warehouses.Warehouse
	events    struct {
		sync.Mutex
		queue [][]any
	}
}

// newStore returns a new Store for the workspace ws.
func newStore(ws *state.Workspace) (*Store, error) {
	store := &Store{
		workspace: ws.ID,
	}
	var err error
	store.redis, _, err = openRedis(ws.Redis.Settings)
	if err != nil {
		return nil, fmt.Errorf("cannot open Redis database: %s", err)
	}
	store.warehouse, err = openWarehouse(ws.Warehouse.Type, ws.Warehouse.Settings)
	if err != nil {
		err2 := store.redis.Close()
		if err != nil {
			// TODO(marco): write the error into a workspace specific log
			log.Printf("[error] error occurred closing Redis database: %s", err2)
		}
		return nil, fmt.Errorf("cannot open data warehouse: %s", err)
	}
	go func() {
		ticker := time.NewTicker(flushEventsQueueTimeout)
		for {
			select {
			case <-ticker.C:
				store.events.Lock()
				events := store.events.queue
				store.events.queue = nil
				store.events.Unlock()
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
	store.events.Lock()
	store.events.queue = append(store.events.queue, events...)
	store.events.Unlock()
}

// CreateUser creates a user with the given properties and returns its
// identifier.
func (store *Store) CreateUser(ctx context.Context, user map[string]any) (int, error) {
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
	err = store.setRedisUserIndex(ctx, id, user)
	if err != nil {
		return 0, err
	}
	err = store.redis.Save(ctx).Err()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// DeleteUser deletes the user with identifier id.
func (store *Store) DeleteUser(ctx context.Context, id int) error {
	err := store.deleteRedisUserIndex(ctx, id)
	if err != nil {
		return err
	}
	_, err = store.warehouse.Exec(ctx, "DELETE FROM `users` WHERE `id` = ?", id)
	return err
}

// DestinationUser returns the external ID of the destination user of the
// action that matches with the corresponding property. If it cannot be
// found, then the empty string and false are returned.
func (store *Store) DestinationUser(ctx context.Context, action int, property string) (string, bool, error) {
	return store.warehouse.DestinationUser(ctx, action, property)
}

// Events returns the events that satisfy the where condition with only the
// given columns, ordered by order if order is not the zero Property, and in
// range [first,first+limit] with first >= 0 and 0 < limit <= 1000.
//
// If a query to the warehouse fails, it returns an Error value.
func (store *Store) Events(ctx context.Context, columns []types.Property, where Where, order types.Property, first, limit int) ([][]any, error) {
	return store.warehouse.Select(ctx, "events", columns, where, order, first, limit)
}

// FilterCandidatesByProperty filters candidate users based on the specified
// property during identity resolution. It returns the users in candidates
// who have the given value for the property. If the value argument is nil, it
// returns users that have the zero value for the property. As a special case,
// if the candidates argument is nil, it means that candidates contain every
// user.
//
// property must be an anonymous identifier or an identifier for an action.
func (store *Store) FilterCandidatesByProperty(ctx context.Context, candidates []int, property string, value any) ([]int, error) {
	var key string
	if value == nil {
		key = redisPropsZeroKey(property)
	} else {
		key = redisPropsKey(property, value)
	}
	var users []int
	err := store.redis.LRange(ctx, key, 0, -1).ScanSlice(&users)
	if err != nil {
		return nil, err
	}
	if candidates != nil {
		users = intersection(users, candidates)
	}
	return users, nil
}

// InitWarehouse initializes the data warehouse creating the events and the
// destinations_users tables.
func (store *Store) InitWarehouse(ctx context.Context) error {
	return store.warehouse.Init(ctx)
}

// SetDestinationUser sets the destination user relative to the action, with
// the given external user ID and external property.
func (store *Store) SetDestinationUser(ctx context.Context, connection int, externalUserID, externalProperty string) error {
	return store.warehouse.SetDestinationUser(ctx, connection, externalUserID, externalProperty)
}

// Schemas returns the schemas of users and groups for the relative tables.
// If a table does not exist, it returns the invalid schema for that table.
func (store *Store) Schemas(ctx context.Context) (types.Type, types.Type, error) {
	tables, err := store.warehouse.Tables(ctx)
	if err != nil {
		return types.Type{}, types.Type{}, err
	}
	var usersSchema, groupsSchema types.Type
	for _, table := range tables {
		if table.Name != "users" && table.Name != "groups" {
			continue
		}
		properties, err := ColumnsToProperties(table.Columns)
		if err != nil {
			return types.Type{}, types.Type{}, err
		}
		if table.Name == "users" {
			usersSchema = types.Object(properties)
		} else {
			groupsSchema = types.Object(properties)
		}
	}
	return usersSchema, groupsSchema, nil
}

// UpdateUser updates the properties of the user with identifier id.
// Only the properties in users will be updated.
func (store *Store) UpdateUser(ctx context.Context, id int, user map[string]any) error {
	// Since the user contains only the properties to update, retrieve every
	// property of the user, then update its index on Redis.
	redisUser, err := store.User(ctx, id)
	if err != nil {
		return err
	}
	maps.Copy(redisUser, user)
	err = store.deleteRedisUserIndex(ctx, id)
	if err != nil {
		return err
	}
	err = store.setRedisUserIndex(ctx, id, redisUser)
	if err != nil {
		return err
	}
	// TODO(Gianluca): find a better way to implement persistency.
	// See https://github.com/open2b/chichi/issues/215.
	err = store.redis.Save(ctx).Err()
	if err != nil {
		return err
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
	values = append(values, id)
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

// User returns the non-zero property values of the user, with the given GID, as
// a map[string]any.
// TODO: revise this method. Investigate on change or remove this.
// See the issue https://github.com/open2b/chichi/issues/243.
func (store *Store) User(ctx context.Context, id int) (map[string]any, error) {
	// Retrieve the keys for the user.
	key := redisUserPropsKeysKey(id)
	var keys []string
	err := store.redis.LRange(ctx, key, 0, -1).ScanSlice(&keys)
	if err != nil {
		return nil, err
	}
	// Extract the property values from the keys.
	user := map[string]any{}
	for _, key := range keys {
		key = strings.TrimPrefix(key, "props:")
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 {
			return nil, errors.New("malformed key")
		}
		property := parts[0]
		rawPropValue := strings.TrimPrefix(key, property+":")
		if rawPropValue == "-" {
			continue
		}
		v := redisJSONDeserialize(rawPropValue)
		user[property] = v
	}
	return user, nil
}

// Users returns the users that satisfy the where condition with only the given
// properties, ordered by order if order is not the zero Property, and in range
// [first,first+limit] with first >= 0 and 0 < limit <= 1000.
//
// If a query fails, it returns an Error value.
func (store *Store) Users(ctx context.Context, properties []types.Property, where Where, order types.Property, first, limit int) ([]map[string]any, error) {
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
func (store *Store) UsersSlice(ctx context.Context, properties []types.Property, where Where, order types.Property, first, limit int) ([][]any, error) {
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

// close closes the store.
// It flushes the events and closes the Redis database and the data warehouse.
func (store *Store) close() error {
	store.events.Lock()
	if len(store.events.queue) > 0 {
		store.flushEvents(store.events.queue)
		store.events.queue = nil
	}
	err := store.redis.Close()
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
	store.events.Unlock()
	return err
}

// deleteRedisUserIndex deletes from the index the user with the given GID.
func (store *Store) deleteRedisUserIndex(ctx context.Context, id int) error {
	// Remove the user from the keys "props:<property>:<property value>" and
	// "props:<property>:-".
	key := redisUserPropsKeysKey(id)
	var keys []string
	err := store.redis.LRange(ctx, key, 0, -1).ScanSlice(&keys)
	if err != nil {
		return err
	}
	for _, key := range keys {
		err := store.redis.LRem(ctx, key, 0, id).Err()
		if err != nil {
			return err
		}
	}
	// Delete the key "user_prop_keys:<user GID>".
	err = store.redis.Del(ctx, key).Err()
	return err
}

func (store *Store) setRedisUserIndex(ctx context.Context, id int, user map[string]any) error {
	// Write the user properties to the keys "props:<property>:<property value>"
	// and "props:<property>:-".
	userPropKeys := make([]any, 0, len(user))
	for p, v := range user {
		var key string
		if zero(v) {
			key = redisPropsZeroKey(p)
		} else {
			key = redisPropsKey(p, v)
		}
		err := store.redis.LPush(ctx, key, id).Err()
		if err != nil {
			return err
		}
		userPropKeys = append(userPropKeys, key)
	}
	// Push the user's properties keys to "user_prop_keys:<user GID>".
	key := redisUserPropsKeysKey(id)
	err := store.redis.LPush(ctx, key, userPropKeys...).Err()
	return err
}

// intersection returns the intersection between a and b.
// The elements in the returned slice are ordered as they appear in a.
func intersection[T comparable](a, b []T) []T {
	out := []T{}
	for _, v := range a {
		for _, w := range b {
			if w == v {
				out = append(out, v)
				break
			}
		}
	}
	return out
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

func redisPropsKey(property string, value any) string {
	v := redisJSONSerialize(value)
	return fmt.Sprintf("props:%s:%s", property, v)
}

func redisUserPropsKeysKey(gid int) string {
	return fmt.Sprintf("user_prop_keys:%d", gid)
}

func redisPropsZeroKey(property string) string {
	return fmt.Sprintf("props:%s:-", property)
}

func zero(v any) bool {
	return v == "" || v == 0 || v == nil
}
