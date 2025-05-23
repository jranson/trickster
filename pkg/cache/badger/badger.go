/*
 * Copyright 2018 The Trickster Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package badger is the BadgerDB implementation of the Trickster Cache
package badger

import (
	"time"

	"github.com/trickstercache/trickster/v2/pkg/cache"
	"github.com/trickstercache/trickster/v2/pkg/cache/metrics"
	"github.com/trickstercache/trickster/v2/pkg/cache/options"
	"github.com/trickstercache/trickster/v2/pkg/cache/status"
	"github.com/trickstercache/trickster/v2/pkg/locks"
	"github.com/trickstercache/trickster/v2/pkg/observability/logging"
	"github.com/trickstercache/trickster/v2/pkg/observability/logging/logger"

	"github.com/dgraph-io/badger"
)

// Cache describes a Badger Cache
type Cache struct {
	Name   string
	Config *options.Options
	locker locks.NamedLocker

	dbh *badger.DB
}

// Locker returns the cache's locker
func (c *Cache) Locker() locks.NamedLocker {
	return c.locker
}

// SetLocker sets the cache's locker
func (c *Cache) SetLocker(l locks.NamedLocker) {
	c.locker = l
}

// Configuration returns the Configuration for the Cache object
func (c *Cache) Configuration() *options.Options {
	return c.Config
}

// Connect opens the configured Badger key-value store
func (c *Cache) Connect() error {
	logger.Info("badger cache setup", logging.Pairs{"cacheDir": c.Config.Badger.Directory})

	opts := badger.DefaultOptions(c.Config.Badger.Directory)
	opts.ValueDir = c.Config.Badger.ValueDirectory

	var err error
	c.dbh, err = badger.Open(opts)
	if err != nil {
		return err
	}

	return nil
}

// Store places the the data into the Badger Cache using the provided Key and TTL
func (c *Cache) Store(cacheKey string, data []byte, ttl time.Duration) error {
	metrics.ObserveCacheOperation(c.Name, c.Config.Provider, "set", "none", float64(len(data)))
	logger.Debug("badger cache store", logging.Pairs{"key": cacheKey, "ttl": ttl})
	return c.dbh.Update(func(txn *badger.Txn) error {
		return txn.SetEntry(&badger.Entry{Key: []byte(cacheKey), Value: data, ExpiresAt: uint64(time.Now().Add(ttl).Unix())}) // #nosec G115 - assume time values are positive
	})
}

// Retrieve gets data from the Badger Cache using the provided Key
// because Badger manages Object Expiration internally, allowExpired is not used.
func (c *Cache) Retrieve(cacheKey string, _ bool) ([]byte, status.LookupStatus, error) {
	var data []byte
	err := c.dbh.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(cacheKey))
		if err != nil {
			return err
		}
		data, err = item.ValueCopy(nil)
		return err

	})

	if err == nil {
		logger.Debug("badger cache retrieve", logging.Pairs{"key": cacheKey})
		metrics.ObserveCacheOperation(c.Name, c.Config.Provider, "get", "hit", float64(len(data)))
		return data, status.LookupStatusHit, nil
	}

	if err == badger.ErrKeyNotFound {
		err = cache.ErrKNF
		logger.Debug("badger cache miss", logging.Pairs{"key": cacheKey})
		metrics.ObserveCacheMiss(c.Name, c.Config.Provider)
		return nil, status.LookupStatusKeyMiss, err
	}

	logger.Debug("badger cache retrieve failed", logging.Pairs{"key": cacheKey, "reason": err.Error()})
	metrics.ObserveCacheMiss(c.Name, c.Config.Provider)
	return data, status.LookupStatusError, err
}

// Remove removes an object in cache, if present
func (c *Cache) Remove(cacheKey string) {
	logger.Debug("badger cache remove", logging.Pairs{"key": cacheKey})
	c.dbh.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(cacheKey))
	})
	metrics.ObserveCacheDel(c.Name, c.Config.Provider, 0)
}

// BulkRemove removes a list of objects from the cache. noLock is not used for Badger
func (c *Cache) BulkRemove(cacheKeys []string) {
	logger.Debug("badger cache bulk remove", nil)

	c.dbh.Update(func(txn *badger.Txn) error {
		for _, key := range cacheKeys {
			if err := txn.Delete([]byte(key)); err != nil {
				return err
			}
			metrics.ObserveCacheDel(c.Name, c.Config.Provider, 0)
		}
		return nil
	})
}

// Close closes the Badger Cache
func (c *Cache) Close() error {
	return c.dbh.Close()
}

// SetTTL updates the TTL for the provided cache object
func (c *Cache) SetTTL(cacheKey string, ttl time.Duration) {
	var data []byte
	err := c.dbh.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(cacheKey))
		if err != nil {
			return nil
		}
		data, _ = item.ValueCopy(nil)
		return txn.SetEntry(&badger.Entry{Key: []byte(cacheKey), Value: data, ExpiresAt: uint64(time.Now().Add(ttl).Unix())}) // #nosec G115 - assume time values are positive
	})
	logger.Debug("badger cache update-ttl", logging.Pairs{"key": cacheKey, "ttl": ttl, "success": err == nil})
	if err == nil {
		metrics.ObserveCacheOperation(c.Name, c.Config.Provider, "update-ttl", "none", 0)
	}
}

func (c *Cache) GetExpires(cacheKey string) (uint64, error) {
	var expires uint64
	err := c.dbh.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(cacheKey))
		if err != nil {
			return err
		}
		expires = item.ExpiresAt()
		return nil
	})
	return expires, err
}
