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

package errors

import "errors"

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrInvalidCredentialsFormat = errors.New("invalid credentials format")
var ErrInvalidName = errors.New("invalid authenticator name")
var ErrInvalidProvider = errors.New("invalid authenticator provider name")
var ErrInvalidUsersFile = errors.New("users does not exist or is not readable")
