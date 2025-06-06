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

package options

import (
	"strings"

	"github.com/trickstercache/trickster/v2/pkg/util/yamlx"
)

const testYAML = `
backends:
  test_pool_member:
    provider: rp
    origin_url: http://example.com
  test:
    tracing_name: test
    hosts:
      - 1.example.com
    revalidation_factor: 2
    multipart_ranges_disabled: true
    dearticulate_upstream_ranges: true
    compressible_types:
      - image/png
    provider: test_type
    cache_name: test
    origin_url: 'scheme://test_host/test_path_prefix'
    api_path: test_api_path
    max_idle_conns: 23
    keep_alive_timeout: 7000
    ignore_caching_headers: true
    timeseries_retention_factor: 666
    timeseries_eviction_method: lru
    fast_forward_disable: true
    backfill_tolerance: 301000ms
    backfill_tolerance_points: 2
    timeout: 37000ms
    timeseries_ttl: 8666000ms
    max_ttl: 300000ms
    fastforward_ttl: 382000ms
    require_tls: true
    max_object_size_bytes: 999
    cache_key_prefix: test-prefix
    path_routing_disabled: false
    forwarded_headers: x
    negative_cache_name: test
    rule_name: ''
    shard_max_size_time: 0ms
    shard_max_size_points: 0
    shard_step: 0ms
    healthcheck:
      headers:
        Authorization: Basic SomeHash
      path: /test/upstream/endpoint
      verb: test_verb
      query: query=1234
    paths:
      series:
        path: /series
        handler: proxy
        req_rewriter_name: ''
      label:
        path: /label
        handler: localresponse
        match_type: prefix
        response_code: 200
        response_body: test
        collapsed_forwarding: basic
        response_headers:
          X-Header-Test: test-value
    prometheus:
      labels:
        testlabel: trickster
    alb:
      methodology: rr
      pool: [ 'test_pool_member' ]
    tls:
      full_chain_cert_path: file.that.should.not.exist.ever.pem
      private_key_path: file.that.should.not.exist.ever.pem
      insecure_skip_verify: true
      certificate_authority_paths:
        - file.that.should.not.exist.ever.pem
      client_key_path: test_client_key
      client_cert_path: test_client_cert

`

func fromTestYAML() (*Options, yamlx.KeyLookup, error) {
	return fromYAML(testYAML, "test")
}

func fromTestYAMLWithDefault() (*Options, yamlx.KeyLookup, error) {
	conf := strings.ReplaceAll(testYAML, "    rule_name: ''", "    rule_name: ''\n    is_default: false")
	return fromYAML(conf, "test")
}

func fromTestYAMLWithReqRewriter() (*Options, yamlx.KeyLookup, error) {
	conf := strings.ReplaceAll(testYAML, "    rule_name: ''", "    rule_name: ''\n    req_rewriter_name: test")
	return fromYAML(conf, "test")
}

func fromTestYAMLWithALB() (*Options, yamlx.KeyLookup, error) {
	conf := strings.ReplaceAll(strings.ReplaceAll(testYAML, "    rule_name: ''", `
    rule_name: ''
    alb:
      output_format: prometheus
      mechanism: tsmerge
        `), "    provider: test_type", "    provider: 'alb'")
	return fromYAML(conf, "test")
}
