local _M = {}
local cjson = require "cjson"

local DEFAULT_AUTH_SERVICE_URL = "http://auth-service:8081"
local DEFAULT_CONFIG_FILE = "/app/auth_config.json"
local DEFAULT_REQUESTS_PER_TOKEN = 100
local DEFAULT_TOKEN_EXPIRY_MINUTES = 10

-- Read and parse JSON configuration file
function _M.read_json_config(file_path)
    local file = io.open(file_path, "r")
    if not file then
        ngx.log(ngx.ERR, "auth_config: Could not open config file: ", file_path)
        return nil
    end

    local content = file:read("*all")
    file:close()

    if not content or content == "" then
        ngx.log(ngx.ERR, "auth_config: Config file is empty: ", file_path)
        return nil
    end

    local ok, config_data = pcall(cjson.decode, content)
    if not ok then
        ngx.log(ngx.ERR, "auth_config: Failed to parse JSON config: ", config_data)
        return nil
    end

    if type(config_data) ~= "table" then
        ngx.log(ngx.ERR, "auth_config: Config is not a JSON object: ", file_path)
        return nil
    end

    return config_data
end

-- Load the configuration. Called from init_by_lua, so the file is read once
-- by the master process and every worker inherits the values.
--
-- The limits must match the ones the auth service runs with, which is why
-- both read the same file.
function _M.init()
    local url = os.getenv("GO_AUTH_SERVICE_URL") or DEFAULT_AUTH_SERVICE_URL
    -- Remove trailing slash to fix proxy_pass behavior
    _M.go_auth_service_url = (url:gsub("/$", ""))

    local config_path = os.getenv("AUTH_CONFIG_FILE") or DEFAULT_CONFIG_FILE
    local config_data = _M.read_json_config(config_path)
    if config_data then
        ngx.log(ngx.NOTICE, "auth_config: Loaded from ", config_path)
    else
        ngx.log(ngx.WARN, "auth_config: Using default limits")
        config_data = {}
    end

    _M.requests_per_token = tonumber(config_data.requests_per_token) or DEFAULT_REQUESTS_PER_TOKEN
    _M.token_expiry_minutes = tonumber(config_data.token_expiry_minutes) or DEFAULT_TOKEN_EXPIRY_MINUTES
end

-- Getter functions for clean access
function _M.get_requests_per_token()
    return _M.requests_per_token
end

function _M.get_token_expiry_minutes()
    return _M.token_expiry_minutes
end

function _M.get_go_auth_service_url()
    return _M.go_auth_service_url
end

return _M
