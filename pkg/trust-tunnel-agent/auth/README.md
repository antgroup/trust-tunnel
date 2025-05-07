# Custom Authorization Plugin

This project allows users to register their own authorization plugins for selecting at runtime.

## How to register your own authorization plugin
Assuming you have an authorization plugin named `myauth`,you need to do the following steps：

1. Implement the `auth.Handler` interface in your code.
2. Call the `RegisterAuthHandlerFactory` function in your code to register your plugin factory function with this project.
3. Specify your plugin name and parameters in your configuration file.
    ```toml
    [auth_config]
    name = "myauth"
    params = {"param1" = "value1","param2" = "value2"}
    ```
4. Declare your plugin in `handler.go` by calling `_ "trust-tunnel/pkg/trust-tunnel-agent/auth/myauth"`
