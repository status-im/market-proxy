-- auth_utils_test.lua
describe("auth_utils", function()
    local auth_utils
    local saved_ngx

    local function with_authorization(value)
        _G.ngx = {
            WARN = "warn",
            log = function() end,
            var = { http_authorization = value },
        }
    end

    setup(function()
        require("spec_helper")
        auth_utils = require("utils.auth_utils")
    end)

    before_each(function()
        saved_ngx = _G.ngx
    end)

    after_each(function()
        _G.ngx = saved_ngx
    end)

    describe("extract_jwt_token", function()
        it("returns the token of a Bearer header", function()
            with_authorization("Bearer abc.def.ghi")

            assert.are.equal("abc.def.ghi", auth_utils.extract_jwt_token())
        end)

        it("returns nothing without an Authorization header", function()
            with_authorization(nil)

            assert.is_nil(auth_utils.extract_jwt_token())
        end)

        it("leaves basic auth credentials to nginx", function()
            with_authorization("Basic dXNlcjpwYXNz")

            assert.is_nil(auth_utils.extract_jwt_token())
        end)

        it("returns nothing for a Bearer header without a token", function()
            with_authorization("Bearer")

            assert.is_nil(auth_utils.extract_jwt_token())
        end)

        it("accepts a token of the maximum length", function()
            local token = string.rep("a", 4096)
            with_authorization("Bearer " .. token)

            assert.are.equal(token, auth_utils.extract_jwt_token())
        end)

        it("rejects a token longer than the maximum", function()
            with_authorization("Bearer " .. string.rep("a", 4097))

            assert.is_nil(auth_utils.extract_jwt_token())
        end)
    end)
end)
