//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

// Package index implements functions and methods for working with indexes on a
// Redis instance.
package index

import (
	"context"
	"encoding/json"
	"log"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// Index is an index on a Redis instance.
type Index struct {
	client *redis.Client
}

// Open opens an index on a Redis client.
func Open(client *redis.Client) *Index {
	return &Index{
		client: client,
	}
}

// UsersByPropertyValue returns the GIDs of the users that have the given
// property with the given value.
func (i *Index) UsersByPropertyValue(ctx context.Context, property string, value any) ([]int, error) {
	// TODO(Gianluca): refactor and/or optimize this.
	key := "props:" + property + ":" + jsonSerialize(value)
	var ids []int
	err := i.client.LRange(ctx, key, 0, -1).ScanSlice(&ids)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// SetUser sets the user gid with the given properties. If an user with the same
// gid already exists in the index, its properties are replaced with the given
// properties; otherwise, if it does not exist, a new user is created in the
// index.
func (i *Index) SetUser(ctx context.Context, gid int, properties map[string]any) error {
	// TODO(Gianluca): refactor and/or optimize this.
	err := i.removeUser(ctx, gid)
	if err != nil {
		return err
	}
	err = i.storeUser(ctx, gid, properties)
	if err != nil {
		return err
	}
	// TODO(Gianluca): find a better way to implement persistency.
	err = i.client.Save(ctx).Err()
	return err
}

func (i *Index) storeUser(ctx context.Context, id int, user map[string]any) error {
	// TODO(Gianluca): refactor and/or optimize this.
	keys := make([]any, 0, len(user))
	for p, v := range user {
		key := "props:" + p + ":" + jsonSerialize(v)
		err := i.client.LPush(ctx, key, id).Err()
		if err != nil {
			return err
		}
		keys = append(keys, key)
	}
	err := i.client.Del(ctx, "userkeys:"+strconv.Itoa(id)).Err()
	if err != nil {
		return err
	}
	err = i.client.LPush(ctx, "userkeys:"+strconv.Itoa(id), keys...).Err()
	if err != nil {
		return err
	}
	return nil
}

func (i *Index) removeUser(ctx context.Context, id int) error {
	// TODO(Gianluca): refactor and/or optimize this.
	key := "userkeys:" + strconv.Itoa(id)
	var keys []string
	err := i.client.LRange(ctx, key, 0, -1).ScanSlice(&keys)
	if err != nil {
		return err
	}
	for _, key := range keys {
		err := i.client.LRem(ctx, key, 0, id).Err()
		if err != nil {
			return err
		}
	}
	err = i.client.Del(ctx, "userkeys:"+strconv.Itoa(id)).Err()
	return err
}

func jsonSerialize(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		log.Panic(err)
	}
	return string(data)
}
