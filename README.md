# last.fm RPC

Discord RPC implementation for [last.fm](last.fm), primarily for my own use. Requires that you have a last.fm API key and that the environmental variable `LASTFM_API_KEY` be set to that API key.

This program can be installed with

```bash
go install github.com/jsdoublel/lastfm-rpc@latest
```

or be build from source.

```bash
git clone https://github.com/jsdoublel/lastfm-rpc.git
cd lastfm-rpc
go build
```

In order to function, the application will need your last.fm username and a last.fm API key. Both of these can be provided in a `lastfm-rpc.toml` file, located in your config file directory (for instance `.config/` on Linux). If you run the application, a template for this file will be created, and its location will be printed.

Once installed and configured, simply run the executable

```bash
lastfm-rpc
```

