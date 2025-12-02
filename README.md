<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="frontend/static/baby_octo.svg">
    <source media="(prefers-color-scheme: light)" srcset="frontend/static/baby_octo_dark.svg">
    <img src="frontend/static/baby_octo_dark.svg" alt="Octobud Logo" width="120" height="120">
  </picture>
</p>

<h1 align="center">Octobud</h1>

<p align="center">
  <strong>Like Gmail, but for your GitHub notifications.</strong>
</p>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#documentation">Documentation</a> •
  <a href="#contributing">Contributing</a>
</p>

<p align="center">
  <a href="https://github.com/ajbeattie/octobud/actions/workflows/ci.yml?query=branch%3Amain"><img src="https://github.com/ajbeattie/octobud/actions/workflows/ci.yml/badge.svg?branch=main" alt="CI Status"></a>
  <a href="LICENSE.txt"><img src="https://img.shields.io/badge/license-AGPL--3.0-blue.svg" alt="License"></a>
  <a href="https://github.com/ajbeattie/octobud"><img src="https://img.shields.io/github/stars/ajbeattie/octobud?style=social" alt="GitHub Stars"></a>
</p>

---

Octobud is a self-hosted web application that helps you manage the flood of GitHub notifications with filtering, custom views, keyboard shortcuts, and automation rules.

Built with **Go**, **Svelte 5**, and **PostgreSQL**.

## Features

### Gmail-inspired Notification Inbox
![Default Inbox View](docs/media/hero_inbox.gif)

### Split Pane Mode
![Split Pane Mode](docs/media/hero_split_detail.gif)

### More

| Feature | Description |
|---------|-------------|
| **Full lifecycle management** | Star, Snooze, Archive, Tag, and Mute notifications |
| **Inline Issue and PR details** | View status, comments, and more inline |
| **Super fast loading** | Issue, PR, and Discussion data is stored locally for fast lookup and filtering |
| **Custom Views** | Create filtered views with a rich query language |
| **Keyboard-First** | Navigate and manage notifications without touching your mouse |
| **Automation Rules** | Automatically archive, filter, or tag notifications based on criteria |
| **Tags** | Organize notifications with custom tags and colors |
| **Real-Time Sync** | Background worker keeps notifications up to date |
| **Desktop notifications** | Never miss a review request or issue reply |
| **Privacy-First** | Self-hosted, your data stays with you |

## Quick Start

**Prerequisites:** [Docker](https://docs.docker.com/get-docker/) and a [GitHub PAT](https://github.com/settings/tokens) with `notifications` and `repo` scopes.

```bash
git clone https://github.com/ajbeattie/octobud.git && cd octobud
cp .env.example .env
```

Edit `.env` and set `GH_TOKEN` to your GitHub token, then:

```bash
make docker-up
```

🎉 That's it! Open `http://localhost:3000` and login with `octobud`/`octobud`.

> [!IMPORTANT]
> Change your credentials after first login (profile avatar → "Update credentials").

<details>
<summary><strong>More options</strong></summary>

### Dev Build (Hot Reload)

```bash
make docker-up-dev  # Runs on port 5173
```

### With 1Password

Requires the 1Password desktop app, 1Password CLI, and a [configured](https://developer.1password.com/docs/cli/shell-plugins/github/) GitHub PAT.

```bash
OP_GH_TOKEN='op://Vault/Item/token' make docker-up-1password
```

Where `OP_GH_TOKEN` is the path within 1Password to the token.

### Environment Variables

The `.env` file supports these variables:

| Variable | Required | Description |
|----------|----------|-------------|
| `GH_TOKEN` | Yes | GitHub Personal Access Token |
| `JWT_SECRET` | No | Auto-generated if not set |
| `JWT_EXPIRY` | No | Token expiration (default: `168h` = 7 days) |

### Stop Services

```bash
make docker-down
```

### Local Development (without Docker)

For active development with hot reload, see the [Contributing Guide](docs/CONTRIBUTING.md#development-setup).

</details>

## After Setup

1. **Initial Sync** - You'll be prompted to configure how far back to sync notifications. Start small (30 days) - you can always [sync more later](docs/start-here.md#syncing-more-later).

2. **Learn the basics** - Press `h` for keyboard shortcuts, or see [Start Here](docs/start-here.md).

## Documentation

- [Start Here](docs/start-here.md) - Initial setup and core workflows
- [Query Syntax](docs/guides/query-syntax.md) - Filter and search notifications
- [Keyboard Shortcuts](docs/guides/keyboard-shortcuts.md) - Navigate efficiently
- [Views & Rules](docs/guides/views-and-rules.md) - Organize your inbox
- [Concepts](docs/concepts/) - How syncing, queries, and actions work

## Roadmap

See [ROADMAP.md](ROADMAP.md) for planned features.

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](docs/CONTRIBUTING.md) for guidelines.

- 🐛 [Report bugs](https://github.com/ajbeattie/octobud/issues/new?template=bug_report.md)
- 💡 [Request features](https://github.com/ajbeattie/octobud/issues/new?template=feature_request.md)
- 📖 [Improve documentation](https://github.com/ajbeattie/octobud/issues/new?template=docs.md)

## Security

Found a security vulnerability? Please report it responsibly. See [SECURITY.md](SECURITY.md) for details.

## License

Octobud is licensed under the [GNU Affero General Public License v3.0](LICENSE.txt) (AGPL-3.0).

This means you're free to use, modify, and distribute this software, but if you run a modified version as a network service, you must make the source code available to users of that service.

## Disclaimer

Octobud is an independent, open-source project. It is **not affiliated with, endorsed by, or officially supported by GitHub, Inc.** GitHub and the GitHub logo are trademarks of GitHub, Inc.

---

<p align="center">
  Made with ❤️ by someone who hates reading GitHub notifications in plain text and managing a million Gmail filters.
</p>
