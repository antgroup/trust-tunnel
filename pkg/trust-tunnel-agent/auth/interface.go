// Copyright The TrustTunnel Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import "trust-tunnel/pkg/trust-tunnel-agent/backend/request"

type Code int

const (
	InternalServerErr Code = 500
	Forbidden         Code = 403
	Success           Code = 200
	BadRequest        Code = 400
)

type Response struct {
	Code   Code   `json:"code"`
	ErrMsg string `json:"err_msg"`
}

// Handler defines common methods of auth handler.
type Handler interface {
	// VerifyAccessPermission is used to verify the access permissions for the user to the target.
	// req: the permission details to check for the user.
	VerifyAccessPermission(req *request.Info) Response
}
