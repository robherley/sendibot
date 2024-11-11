# sendibot

Subscribe to Sendico updates via Discord.

## Development

1. Set `DISCORD_TOKEN` env var.
2. Need a writeable volume to track subscriptions and updates in SQLite. By default `./sendico.db` is created.
3. Build: `go build`
4. Run: `./sendibot` (or `./sendibot -help` for options)

## Commands

### `/ping`

Pings the bot!

![ping example](docs/img/ping.png)

### `/subscribe`

Subscribe to a search term and shops.

![subscribe term example](docs/img/subscribe.png)
![subscribe shops example](docs/img/subscribe-shops.png)

### `/unsubscribe`

Unsubscribe from search terms(s).

![unsubscribe example](docs/img/unsubscribe.png)

### `/subscriptions`

View active subscriptions.

![subscriptions example](docs/img/subscriptions.png)
