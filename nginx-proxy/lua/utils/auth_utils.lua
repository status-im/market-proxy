local _M = {}

-- JWT tokens are typically < 2KB; longer values are rejected to bound the
-- size of the shared dict keys built from them
local MAX_TOKEN_LENGTH = 4096

-- Extract the JWT token from the Authorization header.
--
-- Unlike eth-rpc-proxy, a token in a query parameter is not accepted: here the
-- query string is the request itself and the proxy cache key, so a token in
-- it would reach the backend and split the cache per client.
function _M.extract_jwt_token()
    local auth_header = ngx.var.http_authorization
    if not auth_header then
        return nil
    end

    local auth_type, token = auth_header:match("^(%S+)%s+(.+)$")
    if auth_type ~= "Bearer" or not token then
        return nil
    end

    if #token > MAX_TOKEN_LENGTH then
        ngx.log(ngx.WARN, "Token too long, rejecting: ", #token, " bytes")
        return nil
    end

    return token
end

return _M
