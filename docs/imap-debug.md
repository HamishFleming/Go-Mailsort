# IMAP Debugging Toolkit

This project now includes an `imap-debug` command to inspect IMAP connectivity and server behavior.

## CLI Commands

Run commands with:

```bash
mailsort imap-debug <subcommand> [args]
```

Available subcommands:

- `capabilities` - list server CAPABILITY values
- `list [pattern]` - list mailboxes (default pattern `*`)
- `select [mailbox]` - select mailbox and print counts (default `INBOX`)
- `fetch-flags <seqset>` - fetch flags for messages
- `fetch-headers <seqset>` - fetch envelope + flags for messages
- `search [mode] [args]` - run simple searches
  - `ALL`
  - `UNSEEN`
  - `SINCE <date>` where date format is `1-Feb-2026`
  - `KEYWORD <flag>`
- `store <seqset> <+|-> <flags...>` - add/remove flags/keywords

## Examples

```bash
mailsort imap-debug capabilities
mailsort imap-debug list "*"
mailsort imap-debug select INBOX
mailsort imap-debug fetch-flags 1:10
mailsort imap-debug fetch-headers 1:10
mailsort imap-debug search ALL
mailsort imap-debug search UNSEEN
mailsort imap-debug search SINCE 1-Feb-2026
mailsort imap-debug search KEYWORD Work
mailsort imap-debug store 1 + Work Important
mailsort imap-debug store 1 - Important
```

## Raw IMAP Session Reference

Useful for manual debugging with `openssl` or `nc`.

```bash
openssl s_client -connect mail.example.com:993 -crlf
```

```text
a1 LOGIN user@example.com password
a2 CAPABILITY
a3 LIST "" "*"
a4 SELECT INBOX
a5 FETCH 1:10 (FLAGS INTERNALDATE BODY.PEEK[HEADER.FIELDS (SUBJECT FROM DATE)])
a6 STORE 1 +FLAGS (Work Important)
a7 STORE 1 -FLAGS (Important)
a8 SEARCH ALL
a9 SEARCH UNSEEN
a10 SEARCH SINCE 1-Feb-2026
a11 SEARCH KEYWORD Work
a99 LOGOUT
```

## Environment Variables

`imap-debug` uses:

- `IMAP_HOST` (default: `imap.mail.yahoo.com`)
- `IMAP_USER`
- `IMAP_PASS`

TLS is enabled by default and port `993` is used.
