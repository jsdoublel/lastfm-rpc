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

Once installed, simply run it, giving it your last.fm username

```bash
lastfm-rpc -u <username>
```
