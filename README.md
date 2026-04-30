# motd

> A message-of-the-day generator that's half facts, half feelings.

Like `fastfetch`, but it tells you that **your VPN is down**, **your scrub is overdue**, and **you have 12 packages to upgrade** — instead of telling you which kernel you're running for the seven-thousandth time. With a friendly animal.

```
    _y~~~~__   ____           host  cube
   ~.      `````   `~_        up    4d 12h 31m
  ~                  :~       load  0.42  0.38  0.31
 F                     L      disk  97G / 888G  (10%)
 :                     F            204G / 32T  (0%)
   _yyy`          _    F      mem   8G / 16G  (50%)
 `: _y L  .y____y_~y`  L      ip    10.10.10.10  (eth)
 ~_```  : :      F  $  4            100.121.19.125  (tailscale)
   `~` y: :     y~ .F  4
       `~~~     `yyy__y

 ● 2 services running → motd services
 ⚠ 12 packages outdated → motd updates
```

(That's an elephant. There are 36 other animals to choose from.)

## What it does

- **Mark on the left**: a colorful animal of your choosing — pick a different one per host so you know where you logged in
- **Facts on the right**: hostname, uptime, load, disk, memory, IP — all driven by tiny built-in collectors so it's <50ms warm
- **Alerts below**: each module reports a one-liner only when something's worth saying; the line vanishes when it's not
- **Drill-down**: `motd <name>` runs that module's full version (e.g. `motd disk` shows a per-mount breakdown with bars + ZFS datasets)
- **`--watch` mode**: edit the config in one pane, watch motd re-render on save in another
- **Cache where it makes sense**: per-module TTL for things that genuinely don't change often, no cache for things that respond to user action

## Install

### From a release

```bash
curl -L https://github.com/arne/motd/releases/latest/download/motd_Linux_x86_64.tar.gz | tar xz
sudo install -m 0755 motd /usr/local/bin/motd
```

### From source

```bash
go install github.com/arne/motd/cmd/motd@latest
```

That's enough — with no config file, `motd` ships with a built-in default that picks a random animal and shows the standard system facts. To customize, drop a config in place:

```bash
# user config (just for you)
mkdir -p ~/.config/motd
motd example-config > ~/.config/motd/config.yaml

# or system-wide for every user on the box
sudo mkdir -p /etc/xdg/motd
motd example-config | sudo tee /etc/xdg/motd/config.yaml >/dev/null
git clone https://github.com/arne/motd /tmp/motd-src
sudo cp -r /tmp/motd-src/examples/marks /etc/xdg/motd/
```

### Hook it into your shell

```bash
# fish
echo "motd" >> ~/.config/fish/config.fish

# bash
echo "motd" >> ~/.bashrc

# zsh
echo "motd" >> ~/.zshrc
```

## Configuration

Single YAML file. Two halves: **layout** (a recursive tree of stacks) and **modules** (named things that produce text).

```yaml
mark: { file: marks/animals/octopus.ansi }

layout:
  vstack:
    - hstack:
        - mark
        - vstack: [host, up, load, disk, mem, ip]
    - vstack: [services, updates, scrub]

modules:
  host:    { builtin: hostname }
  up:      { builtin: uptime }
  load:    { builtin: load }
  mem:     { builtin: mem }
  ip:      { builtin: ip }
  disk:    { builtin: disk }

  services:
    builtin: services
    list:
      - { name: immich,    http: "http://localhost:2283/api/server/ping" }
      - { name: openwebui, cmd:  "incus exec openwebui -- curl -sf http://localhost:8080/health" }

  updates:
    command: |
      n=$(apt list --upgradable 2>/dev/null | tail -n +2 | wc -l)
      [ "$n" -gt 0 ] && echo "$n packages outdated"
    command_full: "apt list --upgradable"

  scrub:
    command: |
      last=$(zpool status storage 2>/dev/null | grep -oP 'on \K.*$' | head -1)
      if [ -n "$last" ]; then
        days=$(( ( $(date +%s) - $(date -d "$last" +%s) ) / 86400 ))
        if [ "$days" -gt 30 ]; then
          echo "storage scrub last ran ${days}d ago"
        fi
      fi
    command_full: "zpool status -v"
    cache: 6h
```

## Modules

### Built-ins

| Name | What it shows |
|---|---|
| `hostname` | The hostname, in your terminal's primary accent |
| `uptime`   | `4d 12h 31m`, friendly format |
| `load`     | `0.42  0.38  0.31` |
| `mem`      | `8G / 16G  (50%)` from `/proc/meminfo` |
| `ip`       | Non-loopback IPv4 addresses, with friendly category labels (`eth`, `wifi`, `tailscale`, `incus`, `docker`, `wg`, `vpn`, `zerotier`, ...) |
| `disk`     | Root + auto-detected zpool roots; `motd disk` shows per-mount table with utilization bars and ZFS datasets |
| `services` | Heterogeneous health checks: `http`, `tcp`, `unit` (systemd), or `cmd`. `motd services` shows a per-service status table |

### Custom

Anything you can put in a shell command. Output goes through the styling pipeline.

```yaml
weather:
  command: "curl -s wttr.in/?format=3"
  command_full: "curl -s wttr.in/"
  cache: 30m

uptime_kuma:
  command: "~/bin/check-status-page"
```

**Convention** — for command-based modules:
- Empty stdout = silent (no alert renders)
- Any stdout = treated as a warning, framework prepends ⚠ glyph and appends drill-down hint
- Exit nonzero = treat as failure; keep last good cache, don't overwrite
- `command_full` is what `motd <name>` runs — use it to dump the verbose view

## Marks

37 cute animals live in `examples/marks/animals/`. Pick one per host to differentiate where you logged in:

```
bat   bear   bee   butterfly   cat   cow   crab   dog
dolphin   dragon   duck   elephant   fox   frog   giraffe
hamster   hedgehog   koala   lion   llama   monkey   mouse
octopus   otter   owl   panda   penguin   pig   rabbit
raccoon   shark   sloth   tiger   turtle   unicorn   whale   wolf
```

Want one we don't ship? Find its emoji codepoint at [openmoji.org](https://openmoji.org/) and run:

```bash
scripts/make-animal-mark.sh <name> <CODEPOINT>
# e.g. scripts/make-animal-mark.sh chicken 1F414
```

The pipeline: download OpenMoji SVG → render to high-res PNG → trim → chafa-render to colored ASCII at 24×14 cells.

## Drill-down

Each module is also accessible directly:

```bash
motd services    # full status table for all services
motd disk        # per-mount breakdown with bars + ZFS datasets
motd updates     # apt list --upgradable
```

If a module has a `command_full` set, that's what runs. Builtins like `disk` and `services` ship their own drill-down.

## --watch

```bash
motd --watch
```

Re-renders whenever the config or any referenced file changes. Edit YAML in one pane, see the result in another. Ctrl+C to exit. Watches the config file's directory and the mark file's directory; debounces multi-event editor saves.

## Why not fastfetch / cowsay?

|  | fastfetch | cowsay | motd |
|---|:---:|:---:|:---:|
| System facts | ✓ | | ✓ |
| Actionable alerts | | | ✓ |
| Configurable layout | partial | | ✓ |
| Drill-down | | | ✓ |
| Friendly animal | | ✓ | ✓ |
| Re-renders on config save | | | ✓ |
| Useful out-of-the-box | ✓ | ✓ | ✓ |
| Cute, in 50ms | | | ✓ |

## Configuration file lookup

In order:

1. `--config <path>` flag
2. `$MOTD_CONFIG` env var
3. `~/.config/motd/config.yaml` (if it exists)
4. `/etc/xdg/motd/config.yaml` (system fallback)
5. **Built-in default** — used when none of the above resolve. A random animal mark plus host/uptime/load/mem/ip/disk. No config file is created on disk; just run `motd example-config > ~/.config/motd/config.yaml` when you're ready to customize.

If `--config` or `$MOTD_CONFIG` points at a file that doesn't exist, that's an error — the built-in default only kicks in when no path was specified.

Mark file paths in config are resolved relative to the config file's directory, so `marks/animals/elephant.ansi` works whether you're using the system or user config.

## `motd update`

Replaces the running binary with the latest release from GitHub. Verifies the SHA256 against the release's `checksums.txt` before swapping the file in place.

```bash
motd update           # download and install if newer
motd update --check   # report current vs. latest, install nothing
motd update --force   # reinstall even when already on the latest
```

If the binary lives somewhere only root can write (e.g. `/usr/local/bin`), re-run with `sudo`. For Go-installed builds (`go install`), `~/go/bin` is user-writable so no sudo needed.

## `motd version`

Prints the version, commit, and build date. Equivalent to `motd --version`.

## `motd refresh`

Wipes the per-module cache. Run it after an action that invalidates a cached module — most commonly `apt upgrade`, where the `updates` count would otherwise stay stale until its TTL expires.

```bash
sudo apt upgrade && motd refresh
```

For a single-user box, you can wire it into apt with a dpkg hook. The cache is per-user (`~/.cache/motd`), so this only refreshes the cache for the user who invoked `sudo apt`; other users keep their own caches and will see stale counts until their TTL expires or they run `motd refresh` themselves. The hook also no-ops under `unattended-upgrades` and other non-`sudo` apt invocations, since `$SUDO_USER` is unset there. Use the absolute path to your `motd` install — root's `PATH` typically doesn't include `~/go/bin`.

```bash
sudo tee /etc/apt/apt.conf.d/99-motd-refresh <<'EOF'
DPkg::Post-Invoke { "[ -n \"$SUDO_USER\" ] && sudo -u \"$SUDO_USER\" /usr/local/bin/motd refresh >/dev/null 2>&1 || true"; };
EOF
```

## `motd example-config`

Prints a sample config to stdout. Pipe it into a file and edit from there:

```bash
mkdir -p ~/.config/motd
motd example-config > ~/.config/motd/config.yaml
```

## Building from source

```bash
git clone https://github.com/arne/motd
cd motd
go build -o motd ./cmd/motd
./motd --config examples/config.yaml
```

Releases are built via [goreleaser](https://goreleaser.com/) on tag push. See `.goreleaser.yml` for the build matrix.

## License

MIT. See [LICENSE](LICENSE).
