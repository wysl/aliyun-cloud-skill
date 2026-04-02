# Audit notes

This refactored version intentionally keeps only the core skill surface.

## What was removed conceptually

- oversized operator-manual style content in the main skill body
- duplicated helper logic spread across many scripts
- a giant action router pattern

## What to preserve going forward

- one primary entrypoint
- one env parser implementation
- one account path model
- one consistent output style per command

## Extension rule

When adding new Aliyun product support, prefer adding subcommands to `scripts/aliyunctl.go` and shared helpers in the same codebase before creating extra scripts.
