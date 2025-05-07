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

import (
	"fmt"
)

// Config defines the structure for an auth handler's configuration, including its name and parameters.
// Name is the name of the auth handler.
// Params is a key-value pair used to store specific parameters for the auth handler.
type Config struct {
	Name   string            `toml:"name"`
	Params map[string]string `toml:"params"`
}

// HandlerConfig is an interface that defines the configuration for an auth handler.
// It serves as a generic type, allowing different auth handlers to implement various configuration structures.
type HandlerConfig interface{}

// authHandlerFactories is a mapping that stores auth handler factory functions.
// The keys are the names of the auth handlers, and the values are functions that create instances of the corresponding auth handler.
var authHandlerFactories = make(map[string]func(config HandlerConfig) Handler)

// RegisterAuthHandlerFactory registers a factory function for an auth handler.
// name identifies the auth handler.
// factoryFunc is a function that creates an auth handler instance with the specified configuration.
// If the auth handler name is already registered, it panics.
func RegisterAuthHandlerFactory(name string, factoryFunc func(config HandlerConfig) Handler) {
	if _, exists := authHandlerFactories[name]; exists {
		panic("auth handler already registered")
	}

	authHandlerFactories[name] = factoryFunc
}

// CreateAuthHandlerFromConfig creates an auth handler instance based on the provided configuration.
// cfg is a Config instance that contains the auth handler's name and parameters.
// It returns a Handler instance, or an error if the corresponding auth handler cannot be found.
func CreateAuthHandlerFromConfig(cfg Config) (Handler, error) {
	factoryFunc, exists := authHandlerFactories[cfg.Name]
	if !exists {
		return nil, fmt.Errorf("authorization handler not found: %s", cfg.Name)
	}

	return factoryFunc(cfg.Params), nil
}
