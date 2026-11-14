-- token_limit_test.lua
describe("token_limit", function()
    local token_limit

    setup(function()
        require("spec_helper")
        token_limit = require("auth.token_limit")
    end)

    describe("decide", function()
        it("lets the first request of a token through", function()
            assert.are.same({
                status = 200,
                headers = { ["X-RateLimit-Limit"] = "3", ["X-RateLimit-Remaining"] = "2" },
            }, token_limit.decide(1, 3))
        end)

        it("lets the request that reaches the limit through with none remaining", function()
            local answer = token_limit.decide(3, 3)
            assert.are.equal(200, answer.status)
            assert.are.equal("0", answer.headers["X-RateLimit-Remaining"])
        end)

        it("rejects the request past the limit", function()
            assert.are.same({
                status = 401,
                headers = { ["X-RateLimit-Limit"] = "3", ["X-RateLimit-Remaining"] = "0" },
            }, token_limit.decide(4, 3))
        end)

        it("never reports a negative remainder", function()
            assert.are.equal("0", token_limit.decide(10, 3).headers["X-RateLimit-Remaining"])
        end)
    end)
end)
