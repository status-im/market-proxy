-- auth_config_test.lua
describe("auth_config", function()
    local auth_config
    local saved_getenv
    local temp_files

    -- Writes content to a temporary file and points AUTH_CONFIG_FILE at it
    local function with_config_file(content, env)
        local path = os.tmpname()
        local file = assert(io.open(path, "w"))
        file:write(content)
        file:close()
        temp_files[#temp_files + 1] = path

        env = env or {}
        env.AUTH_CONFIG_FILE = path
        os.getenv = function(name) return env[name] end
    end

    setup(function()
        require("spec_helper")
        auth_config = require("auth.auth_config")
    end)

    before_each(function()
        saved_getenv = os.getenv
        temp_files = {}
    end)

    after_each(function()
        os.getenv = saved_getenv
        for _, path in ipairs(temp_files) do
            os.remove(path)
        end
    end)

    it("takes the limits from the config file", function()
        with_config_file('{"requests_per_token": 7, "token_expiry_minutes": 3}')

        auth_config.init()

        assert.are.equal(7, auth_config.get_requests_per_token())
        assert.are.equal(3, auth_config.get_token_expiry_minutes())
    end)

    it("falls back to the defaults for limits the file does not set", function()
        with_config_file('{"jwt_secret": "s"}')

        auth_config.init()

        assert.are.equal(100, auth_config.get_requests_per_token())
        assert.are.equal(10, auth_config.get_token_expiry_minutes())
    end)

    it("falls back to the defaults when the file is missing", function()
        os.getenv = function(name)
            if name == "AUTH_CONFIG_FILE" then return "/nonexistent/auth_config.json" end
        end

        auth_config.init()

        assert.are.equal(100, auth_config.get_requests_per_token())
        assert.are.equal(10, auth_config.get_token_expiry_minutes())
    end)

    it("falls back to the defaults when the file is not valid JSON", function()
        with_config_file("{not json")

        auth_config.init()

        assert.are.equal(100, auth_config.get_requests_per_token())
        assert.are.equal(10, auth_config.get_token_expiry_minutes())
    end)

    it("falls back to the defaults when the file is not a JSON object", function()
        with_config_file("42")

        auth_config.init()

        assert.are.equal(100, auth_config.get_requests_per_token())
    end)

    it("uses the default auth service URL when none is configured", function()
        with_config_file("{}")

        auth_config.init()

        assert.are.equal("http://auth-service:8081", auth_config.get_go_auth_service_url())
    end)

    it("strips the trailing slash of the configured auth service URL", function()
        with_config_file("{}", { GO_AUTH_SERVICE_URL = "http://auth:9000/" })

        auth_config.init()

        assert.are.equal("http://auth:9000", auth_config.get_go_auth_service_url())
    end)
end)
