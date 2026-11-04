# OAuth setup — notes for T-010

## Status

Reference for `T-010`. The credentials exist. The real consent flow is not run yet.

## Location

| Item | Value |
|------|-------|
| Client secret | `/Users/olegk/Documents/_docs/_dev/_horizont/horizont-tools/apps/.secrets/gws/config/client_secret.json` |
| Encrypted gws credentials | Same `gws/config/` directory |
| OAuth client type | `installed` (desktop) |
| Redirect URI | `http://localhost` |
| Project | `giigii-app` |
| Current gws auth | `gws auth status` reports `none` in the default config directory |

The file contains `client_id`, `client_secret`, `auth_uri`, `token_uri` and
`redirect_uris`. Those values stay outside this repository. Do not copy them into a doc,
a test fixture, a log or a commit (`C-2`, `H-2`).

## Scopes

`T-010` needs only:

```
https://www.googleapis.com/auth/youtube.readonly
```

The client file does not list scopes. The scope is requested in the authorization URL.

## Redirect

The client declares `http://localhost` without a port. The local flow uses a loopback
listener on an ephemeral port. Google requires an exact redirect match. Before the real
consent test, confirm that the client accepts `http://localhost:<port>` or add the exact
callback URI in the Google Cloud console.

This is the one open setup check for `T-010`.

## Environment

Do not export the secret path as a committed value. Use it at runtime:

```sh
GOOGLE_OAUTH_CLIENT_FILE="/Users/olegk/Documents/_docs/_dev/_horizont/horizont-tools/apps/.secrets/gws/config/client_secret.json" \
  ./ytfeed onboard
```

The product reads the path from configuration. The path itself is not a secret. The file
is.

## Security

- The repository ignores `.secrets/` and `.env` (`C-2`).
- The token file is written with `0600` (`C-17`).
- The token and the client secret never appear in logs, traces or errors (`L-10`, `E-17`).
- The test suite uses a fake OAuth server. It never reads the real client secret.

## Preconditions for T-010

1. The YouTube Data API is enabled for `giigii-app`.
2. The OAuth client accepts the loopback callback URI.
3. The owner can complete a browser consent when the callback is running.

If item 1 or 2 fails, `T-010` stays blocked and the failure is recorded. Do not work
around it with a different client.
