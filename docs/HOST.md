# Hosts

Each `[[host]]` entry in `config.toml` defines a directory that gets copied
into `public/hosts/<output>` when the site is built with `make all`.

For Node-based submodules, point `source` at the submodule's `dist/`
directory (e.g. `llm-from-scratch-reader`). `dist/` is gitignored, so on a
fresh clone you must build the submodule first:

```bash
cd abcdlsj.github.io/submodules/llm-from-scratch-reader
npm ci
npm run build
```

Then build the site from the repo root:

```bash
make all
```
