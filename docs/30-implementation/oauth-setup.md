# OAuth setup — notes for T-010

## Status

`T-010` implemented on 2026-11-04. The flow uses PKCE S256, the
`youtube.readonly` scope and a loopback redirect on `127.0.0.1`. The real consent
is the remaining check. `T-010` returns the token. `T-011` stores it.

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

The client declares `http://localhost` without a port. The flow uses
`http://127.0.0.1:<port>` on an ephemeral port. Google treats `127.0.0.1` as a loopback
redirect for a desktop client and accepts any port. No Google Cloud console change is
needed.

This check is resolved. If Google returns `redirect_uri_mismatch`, record the exact error
and do not switch to a different client.

## Environment

Do not export the secret path as a committed value. Use it at runtime:

```sh
GOOGLE_OAUTH_CLIENT_FILE="/Users/olegk/Documents/_docs/_dev/_horizont/horizont-tools/apps/.secrets/gws/config/client_secret.json" \
  go run ./cmd/ytfeed auth login
```

Add `--no-browser` to print the URL instead of opening the browser. The command waits two
minutes for the callback. The path is not a secret. The client file is.

## Security

- The repository ignores `.secrets/` and `.env` (`C-2`).
- `T-011` writes the token file with `0600` (`C-17`). `T-010` does not persist it.
- The token and the client secret never appear in logs, traces or errors (`L-10`, `E-17`).
- The test suite uses a fake OAuth server. It never reads the real client secret.

## Preconditions for T-010

1. The YouTube Data API is enabled for `giigii-app`. Confirmed on 2026-11-04.
2. The OAuth client accepts the loopback callback URI. Confirmed for desktop clients.
3. The owner can complete a browser consent when the callback is running.

If item 3 fails, `T-010` stays blocked and the failure is recorded. Do not work around it
with a different client.

## Consent log

| Date | Result |
|------|--------|
| 2026-11-04 | `ytfeed auth login --no-browser` started. The URL was issued. No callback arrived within two minutes. Exit `2`. |
| 2026-11-04 | Retried with a fresh URL and port. No callback arrived within two minutes. Exit `2`. |

The flow and the callback are ready. The owner must open the URL in a browser on this Mac
and approve the scope within the two-minute window. Then `T-011` can store the token.

## Output

`ytfeed auth login` prints one line:

```
authorized scope=<scope> expires_at=<UTC time> fingerprint=<short hash>
```

It never prints the access token or the refresh token. `T-011` adds the `0600` token
file and the refresh path.
