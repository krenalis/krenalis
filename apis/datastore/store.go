//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package datastore

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"chichi/apis/datastore/warehouses"
	"chichi/apis/state"
	"chichi/connector/types"

	"github.com/redis/go-redis/v9"
)

type (
	Expr   = warehouses.Expr
	Result = warehouses.Result
	Row    = warehouses.Row
	Table  = warehouses.Table
	Where  = warehouses.Where
	Error  = warehouses.Error
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
	store.redis, err = openRedis(ws.Redis.Settings)
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

// AddEvents adds events to the data warehouse.
func (store *Store) AddEvents(events [][]any) {
	store.events.Lock()
	store.events.queue = append(store.events.queue, events...)
	store.events.Unlock()
}

// DeleteUser deletes from the index the user with the given GID.
func (store *Store) DeleteUser(ctx context.Context, id int) error {
	// Remove the user from the keys "props:<property>:<property value>" and
	// "props:<property>:-".
	key := userPropsKeysKey(id)
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

// DestinationUser returns the external ID of the destination user of the
// action that matches with the corresponding property. If it cannot be
// found, then the empty string and false are returned.
func (store *Store) DestinationUser(ctx context.Context, action int, property string) (string, bool, error) {
	return store.warehouse.DestinationUser(ctx, action, property)
}

// Exec executes a query without returning any rows. args are the placeholders.
// If the query fails, it returns an Error value.
func (store *Store) Exec(ctx context.Context, query string, args ...any) (Result, error) {
	return store.warehouse.Exec(ctx, query, args...)
}

// Init initializes the data warehouse by creating the supporting tables.
func (store *Store) Init(ctx context.Context) error {
	return store.warehouse.Init(ctx)
}

// QueryRow executes a query that should return at most one row.
func (store *Store) QueryRow(ctx context.Context, query string, args ...any) Row {
	return store.warehouse.QueryRow(ctx, query, args...)
}

// Select returns the rows from the given table that satisfies the where
// condition with only the given columns, ordered by order if order is not the
// zero Property, and in range [first,first+limit] with first >= 0 and
// 0 < limit <= 1000.
//
// If a query to the warehouse fails, it returns an Error value.
// If an argument is not valid, it panics.
func (store *Store) Select(ctx context.Context, table string, columns []types.Property, where Where, order types.Property, first, limit int) ([][]any, error) {
	return store.warehouse.Select(ctx, table, columns, where, order, first, limit)
}

// SetDestinationUser sets the destination user relative to the action, with
// the given external user ID and external property.
func (store *Store) SetDestinationUser(ctx context.Context, connection int, externalUserID, externalProperty string) error {
	return store.warehouse.SetDestinationUser(ctx, connection, externalUserID, externalProperty)
}

// SetUser sets the user U with the given GID on the index. If an user with the
// same GID already exists in the index, its property values are replaced with
// the property values of U; otherwise, if it does not exist, a new user is
// created in the index.
func (store *Store) SetUser(ctx context.Context, id int, user map[string]any) error {
	err := store.DeleteUser(ctx, id)
	if err != nil {
		return err
	}
	err = store.setUser(ctx, id, user)
	if err != nil {
		return err
	}
	// TODO(Gianluca): find a better way to implement persistency.
	// See https://github.com/open2b/chichi/issues/215.
	err = store.redis.Save(ctx).Err()
	return err
}

// Tables returns the tables of the data warehouse.
// It returns only the tables 'users', 'groups', 'events', and the tables with
// prefix 'users_', 'groups_' and 'events_'.
func (store *Store) Tables(ctx context.Context) ([]*Table, error) {
	return store.warehouse.Tables(ctx)
}

// User returns the non-zero property values of the user, with the given GID,
// as a map[string]any.
func (store *Store) User(ctx context.Context, id int) (map[string]any, error) {
	// Retrieve the keys for the user.
	key := userPropsKeysKey(id)
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
		v := jsonDeserialize(rawPropValue)
		user[property] = v
	}
	return user, nil
}

// UsersByPropertyValue returns the GIDs of the candidates that have the given
// property with the given value.
// A nil value for candidates means every user.
func (store *Store) UsersByPropertyValue(ctx context.Context, candidates []int, property string, value any) ([]int, error) {
	key := propsKey(property, value)
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

// UsersWithNoPropertyValue returns the GIDs of the candidates that have a zero
// value for the given property.
// A nil value for candidates means every user.
func (store *Store) UsersWithNoPropertyValue(ctx context.Context, candidates []int, property string) ([]int, error) {
	key := propsZeroKey(property)
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

func (store *Store) setUser(ctx context.Context, gid int, user map[string]any) error {
	// Write the user properties to the keys "props:<property>:<property value>"
	// and "props:<property>:-".
	userPropKeys := make([]any, 0, len(user))
	for p, v := range user {
		var key string
		if zero(v) {
			key = propsZeroKey(p)
		} else {
			key = propsKey(p, v)
		}
		err := store.redis.LPush(ctx, key, gid).Err()
		if err != nil {
			return err
		}
		userPropKeys = append(userPropKeys, key)
	}
	// Push the user's properties keys to "user_prop_keys:<user GID>".
	key := userPropsKeysKey(gid)
	err := store.redis.LPush(ctx, key, userPropKeys...).Err()
	return err
}
