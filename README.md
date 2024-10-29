# mctui

Manage your minecraft server remotely over https.

**Disclaimer**:
_Some information may be incomplete or incorrect._
_I will improve the documentation over time._

## Problem

- You have a server running on `HostB`
- You need people to manage a minecraft server in `HostB`
- These people should not have ssh access to the machine

**Solution:** Allow uses to manage the server by authenticating using HTTPS.

## Features

- Full RCON support
- Add multiple accounts
- Navigate over command history
- Allow users to create and restore backups using [Tasks](#Tasks)

## Running

There are no setup for the client. Just run the executable with the necessary parameters:

```bash
mctui --host=127.0.0.1 --port=8090
```

### Windows

- You can use the batch files provided to make it easier to execute
- Edit them once to set the appropriate `host` and `port`
- Then, double click to open in your terminal.
- It uses cmd by default.

## Usage

- There is no mouse support
- Login
  - `<tab>` change focus
  - `<return>` login
- Command
  - `<up>` `<down>` run previous commands
  - `<return>` run the command
  - `<C-l>` clear history
  - `<F1>` restore screen (linux only). Equivalent to `!restore`
- Restore
  - `<up>` `<k>` prev line
  - `<down>` `<j>` next line
  - `<left>` `<h>` prev page
  - `<right>` `<l>` next page
  - `/` filter
  - `<esc>` abort

## Tasks

Tasks are special commands that starts with `!`, so the backend can tell the difference from RCON commands, like `/list` or `/kill player`. If setup correctly on [mctui-server](), there are 2 builtin tasks:

- `!backup` make a backups of the curent save
- `!restore` pick a restore point
