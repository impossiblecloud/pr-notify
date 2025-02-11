# pr-notify

Simple tool to notify on PRs waiting for reviews.

## Env variables

The following env variables are required for proper authentication.

Slack app:

- `SLACK_APP_TOKEN="xapp-XXX"` - Slack app token (sensitive)
- `SLACK_BOT_TOKEN="xoxb-XXX"` - Slack bot token (sensitive)

Github app:

- `GITHUB_APP_ID` - Github app ID
- `GITHUB_INSTALLATION_ID` - installation ID (see URL when managing installation)
- `GITHUB_APP_PRIVATE_KEY` - Github app private key PEM (sensitive)
