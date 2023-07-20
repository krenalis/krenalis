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
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/redis/go-redis/v9"
)

// Index is an index on a Redis instance.
//
// The index uses these kinds of keys:
//
//   - props:<property>:<property value>  → [<user GID>, ...]   For properties with non-zero values
//   - props:<property>:-                 → [<user GID>, ...]   For properties with zero values
//   - user_prop_keys:<user GID>          → [<key 1>, ...]      For holding the keys of the properties of an user
type Index struct {
	client *redis.Client
}

// Open opens an index on a Redis client.
func Open(client *redis.Client) *Index {
	return &Index{
		client: client,
	}
}

// DeleteUser deletes from the index the user with the given GID.
func (i *Index) DeleteUser(ctx context.Context, gid int) error {
	// Remove the user from the keys "props:<property>:<property value>" and
	// "props:<property>:-".
	key := userPropsKeysKey(gid)
	var keys []string
	err := i.client.LRange(ctx, key, 0, -1).ScanSlice(&keys)
	if err != nil {
		return err
	}
	for _, key := range keys {
		err := i.client.LRem(ctx, key, 0, gid).Err()
		if err != nil {
			return err
		}
	}
	// Delete the key "user_prop_keys:<user GID>".
	err = i.client.Del(ctx, key).Err()
	return err
}

// GetUser returns the non-zero property values of the user, with the given GID,
// as a map[string]any.
func (i *Index) GetUser(ctx context.Context, gid int) (map[string]any, error) {
	// Retrieve the keys for the user.
	key := userPropsKeysKey(gid)
	var keys []string
	err := i.client.LRange(ctx, key, 0, -1).ScanSlice(&keys)
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

// SetUser sets the user U with the given GID on the index. If an user with the
// same GID already exists in the index, its property values are replaced with
// the property values of U; otherwise, if it does not exist, a new user is
// created in the index.
func (i *Index) SetUser(ctx context.Context, gid int, U map[string]any) error {
	err := i.DeleteUser(ctx, gid)
	if err != nil {
		return err
	}
	err = i.setUser(ctx, gid, U)
	if err != nil {
		return err
	}
	// TODO(Gianluca): find a better way to implement persistency.
	// See https://github.com/open2b/chichi/issues/215.
	err = i.client.Save(ctx).Err()
	return err
}

// UsersByPropertyValue returns the GIDs of the candidates that have the given
// property with the given value.
// A nil value for candidates means every user.
func (i *Index) UsersByPropertyValue(ctx context.Context, candidates []int, property string, value any) ([]int, error) {
	key := propsKey(property, value)
	var users []int
	err := i.client.LRange(ctx, key, 0, -1).ScanSlice(&users)
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
func (i *Index) UsersWithNoPropertyValue(ctx context.Context, candidates []int, property string) ([]int, error) {
	key := propsZeroKey(property)
	var users []int
	err := i.client.LRange(ctx, key, 0, -1).ScanSlice(&users)
	if err != nil {
		return nil, err
	}
	if candidates != nil {
		users = intersection(users, candidates)
	}
	return users, nil
}

func (i *Index) setUser(ctx context.Context, gid int, user map[string]any) error {
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
		err := i.client.LPush(ctx, key, gid).Err()
		if err != nil {
			return err
		}
		userPropKeys = append(userPropKeys, key)
	}
	// Push the user's properties keys to "user_prop_keys:<user GID>".
	key := userPropsKeysKey(gid)
	err := i.client.LPush(ctx, key, userPropKeys...).Err()
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

func jsonDeserialize(raw string) any {
	// TODO: improve or remove this function.
	var v any
	err := json.Unmarshal([]byte(raw), &v)
	if err != nil {
		log.Panic(err)
	}
	return v
}

func jsonSerialize(v any) string {
	// TODO: improve or remove this function.
	data, err := json.Marshal(v)
	if err != nil {
		log.Panic(err)
	}
	return string(data)
}

func zero(v any) bool {
	return v == "" || v == 0 || v == nil
}

// Functions for retrieving index keys.

func propsKey(property string, value any) string {
	v := jsonSerialize(value)
	return fmt.Sprintf("props:%s:%s", property, v)
}

func propsZeroKey(property string) string {
	return fmt.Sprintf("props:%s:-", property)
}

func userPropsKeysKey(gid int) string {
	return fmt.Sprintf("user_prop_keys:%d", gid)
}
