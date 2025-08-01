# AGENTS.md

## General
- Respond in Japanese
- Do only what is instructed.
- Keep responses concise.
- If you are unsure about something, ask for more context.
- DO NOT ASSUME YOU KNOW EVERYTHING, ASK THE USER ABOUT THEIR REASONING.
- Ask users to provide more context (for example imported files etc) when needed.

## Basic Code Style & Guidelines
- Write code comments and PR comments in English.
- Make comments on lines by themselves.
- Format code with `go fmt ./...` or `make fmt`

## Git Usage
- Use only English for branch name
- Use only English for commit messages

## Testing
- Run `make vet` before every commit
- Run `make test` on every commit
- Run `make lint` on every commit
- Run `make vulcheck` on every commit
- Run `make seccheck` on every commit
- Do not create PR if test fails

## Commit Message
- Follow Conventional Commits
