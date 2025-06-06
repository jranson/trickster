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

package mgmt

import "time"

const (
	// DefaultReloadPort is the default port that the Reload endpoint will listen on
	DefaultPort = 8484
	// DefaultReloadAddress is the default address that the Reload endpoint will listen on
	DefaultAddress = "127.0.0.1"
	// DefaultConfigHandlerPath is the default value for the Trickster Config Printout Handler path
	DefaultConfigHandlerPath = "/trickster/config"
	// DefaultPingHandlerPath is the default value for the Trickster Config Ping Handler path
	DefaultPingHandlerPath = "/trickster/ping"
	// DefaultHealthHandlerPath defines the default path for the Health Handler
	DefaultHealthHandlerPath = "/trickster/health"
	// DefaultPurgeByKeyHandlerPath defines the default path for the Cache Purge (by Key) Handler
	DefaultPurgeByKeyHandlerPath = "/trickster/purge/key/"
	// DefaultPurgeByPathHandlerPath defines the default path for the Cache Purge (by Path) Handler
	DefaultPurgeByPathHandlerPath = "/trickster/purge/path/"
	// DefaultPprofServerName defines the default Pprof Server Name
	DefaultPprofServerName = "both"
	// DefaultDrainTimeout is the default time that is allowed for an old configuration's requests to drain
	// before its resources are closed
	DefaultDrainTimeout = 30 * time.Second
	// DefaultRateLimit is the default Rate Limit time for Config Reloads
	DefaultRateLimit = 3 * time.Second
	// DefaultReloadHandlerPath defines the default path for the Reload Handler
	DefaultReloadHandlerPath = "/trickster/config/reload"
)
