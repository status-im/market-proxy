-- spec_helper.lua - makes the nginx Lua modules loadable outside nginx

local current_dir = debug.getinfo(1, "S").source:sub(2):match("(.*)/")
local base_dir = current_dir .. "/.."

package.path = package.path .. ";" ..
  base_dir .. "/lua/?.lua;" ..
  current_dir .. "/?.lua"

-- Minimal ngx global: enough for modules to load and log. Tests that need
-- more replace it with their own fake.
_G.ngx = {
    ERR = "error",
    WARN = "warn",
    NOTICE = "notice",
    log = function() end,
    var = {},
    shared = {},
}
