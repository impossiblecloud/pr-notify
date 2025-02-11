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

## Config

Config example:

```yaml
github_pr_notifications:

  - gh_owner: my-org
    gh_repo: my-repo-1
    gh_pr_labels:
      - enhancement
    gh_pr_include_drafts: true
    gh_pr_ignore_approved: true
    gh_pr_ignore_changes_requested: true
    # Mon-Fri every 2 hours during business hours
    schedule: "CRON_TZ=Europe/Berlin 00 10-18/2 * * 1-5"
    notify:
      slack:
        channel_id: "ABC123000"
        message_header: ":warning: Please review at your earliest convenience @some-group-handle"

  - gh_owner: my-org
    gh_repo: my-repo-2
    gh_pr_labels: []
    # Mon-Fri every 30 hours during business hours
    schedule: "CRON_TZ=Europe/Berlin */30 10-18 * * 1-5"
    notify:
      slack:
        channel_id: "DEF456000"
        message_header: ":warning: Please review at your earliest convenience @some-group-handle"
```
