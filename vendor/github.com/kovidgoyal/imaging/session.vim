" Empty for the moment
lua << EOF
local capabilities = require("cmp_nvim_lsp").default_capabilities()
local lspconfig = require('lspconfig')
lspconfig.gopls.setup({
    capabilities = capabilities,
    settings = {
        gopls = {
            buildFlags = { "-tags=lcms2cgo" },
            directoryFilters = { "-.git", "-bypy/b", "-build", "-dist", }
        }
    }
})
EOF
