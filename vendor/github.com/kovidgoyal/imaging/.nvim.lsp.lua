#!/usr/bin/env lua

local lspconfig = require('lspconfig')

-- Get the existing gopls configuration to merge with it
local gopls_opts = lspconfig.gopls.get_default_options()

-- Merge the project-specific settings
lspconfig.gopls.setup({
    -- Extend or override settings from your global config
    settings = vim.tbl_deep_extend("force", gopls_opts.settings or {}, {
        gopls = {
            buildFlags = { "-tags=lcms2cgo" },
        },
    }),
})
