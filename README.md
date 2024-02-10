## regols

OPA rego language server

![regols](https://user-images.githubusercontent.com/21323222/148948494-d6a59424-d68a-4ab2-8cf4-4759dc9b6316.gif)

## Install

### MacOS

```bash
brew install kitagry/tap/regols
```

### Other

Download from [Release page](https://github.com/kitagry/regols/releases).

Or, you can build as following.

```bash
go install github.com/kitagry/regols@latest
```

## Configuration

### Configuration for [nvim-lspconfig](https://github.com/neovim/nvim-lspconfig)

```vim
local nvim_lsp = require'lspconfig'
local configs = require'lspconfig.configs'

if not configs.regols then
  configs.regols = {
    default_config = {
      cmd = {'regols'};
      filetypes = { 'rego' };
      root_dir = nvim_lsp.util.root_pattern(".git");
    }
  }
end
configs.regols.setup{}
```

## Specs

- [x] textDocument/publishDiagnostics
- [x] textDocument/formatting
- [x] textDocument/definition
- [x] textDocument/completion
- [x] textDocument/hover
