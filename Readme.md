# Chores

## Contribution guidelines

- In the best of our abilities we want to keep the app working without JS. We only use `echarts` for charting. This
  means that all major features work without JS, and we use semantic HTML tags and features.
- We try to limit dependencies as much as possible. We use three go dependencies, a non CGO sqlite3 driver, `/x/crypto`
  and a stdlib extension package. For frontend we use `echarts`.
- SQLC is used for SQL queries. This is a development dependency but not a runtime dependency.
- We use https://tabler.io/icons for icons
- Everything should be bundled in the binary. We use go:embed for static files and also bundle the frontend
  dependencies in the binary.
- Keep it simple, we want to deploy a single binary with no external dependencies (including DB). Code should be
  simple and easy. A bad structure is better than no structure, but a good structure is better than a bad one.
- Limiting dependencies means we the maintenance and complexity burden is lower. The rigid compatability guarantees of
  Go means there will be little churn over time and getting back into the project after a long time is easier.

## Features

- [x] users with auth
    - [x] password auth
    - [ ] other kind of auth methods that don't incur dependencies
    - [ ] user nicknames
- [x] invites
    - [x] to a specific chorelist
- [x] shared chore lists
- [x] chores
    - [x] recurring chores
    - [x] oneshot chores
    - [ ] date chores
    - [ ] date recurring chores (here we need to build some custom language for specifying the recurrence)
    - [x] snoozing
    - [x] expediting (opposite of snoozing)
- [x] insights
    - [x] calendar graph
    - [ ] list member stats

## Roadmap

- improved UI for invite pages (view and accept)
- info popovers triggered by an infoicon, for useronboarding
    - invites
    - password auth
    - invite accept
    - chore types
    - https://developer.mozilla.org/en-US/docs/Web/API/Popover_API/Using
- invites without sharing a chorelist
- setting a longer timeout for invites, 24h is a bit short
- hashed filenames for static files for better caching and no need for css file renaming
- date type chores
- date recurring chores
 