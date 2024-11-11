# sendibot

Subscribe to Sendico updates via Discord.

## Development

1. Set `DISCORD_TOKEN` env var.
2. Need a writeable volume to track subscriptions and updates in SQLite. By default `./sendico.db` is created.
3. Build: `go build`
4. Run: `./sendibot` (or `./sendibot -help` for options)
