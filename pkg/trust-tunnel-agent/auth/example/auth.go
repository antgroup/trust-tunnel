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

package example

import (
	"bytes"
	"encoding/json"
	"net/http"
	"trust-tunnel/pkg/trust-tunnel-agent/auth"
	"trust-tunnel/pkg/trust-tunnel-agent/backend/request"
)

func init() {
	auth.RegisterAuthHandlerFactory("example", func(config auth.HandlerConfig) auth.Handler {
		configMaps := config.(map[string]string)

		return &AuthHandler{
			AuthURL: configMaps["auth_url"],
			Client:  &http.Client{},
		}
	})
}

type AuthHandler struct {
	AuthURL string
	Client  *http.Client
}

func (handler *AuthHandler) VerifyAccessPermission(req *request.Info) auth.Response {
	payloadBytes, err := json.Marshal(req)
	if err != nil {
		return auth.Response{
			Code:   auth.BadRequest,
			ErrMsg: err.Error(),
		}
	}

	// Post the payload to the authentication server.
	resp, err := http.Post(handler.AuthURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return auth.Response{
			Code:   auth.InternalServerErr,
			ErrMsg: err.Error(),
		}
	}
	defer resp.Body.Close()

	// Check the response from the authentication server.
	if resp.StatusCode != http.StatusOK {
		return auth.Response{
			Code:   auth.InternalServerErr,
			ErrMsg: "",
		}
	}

	// Parse the response from the authentication server.
	var authResponse struct {
		Code auth.Code `json:"code"`
	}

	err = json.NewDecoder(resp.Body).Decode(&authResponse)
	if err != nil {
		return auth.Response{
			Code:   auth.InternalServerErr,
			ErrMsg: err.Error(),
		}
	}

	if authResponse.Code != auth.Success {
		return auth.Response{
			Code:   auth.Forbidden,
			ErrMsg: "",
		}
	}

	return auth.Response{
		Code:   auth.Success,
		ErrMsg: "",
	}
}
