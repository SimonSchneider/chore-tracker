# Chores

## Contribution guidelines

- In the best of our abilities we want to keep the app working without JS. We only use HTMX and hyperscript for
  progressive enhancement. This means that all features should work without JS, and we use semantic HTML tags
  and features. We use HTMX and hyperscript to make the app more interactive. Without JS the app can work as MPA
  while HTMX can make it SPA-like.
- We try to limit dependencies as much as possible. We use two go dependencies, a non CGO sqlite3 driver and a
  stdlib extension package. For frontend we use HTMX and hyperscript.
- SQLC is used for SQL queries. This is a development dependency but not a runtime dependency.
- We use https://tabler.io/icons for icons
- Everything should be bundled in the binary. We use go:embed for static files and also bundle the frontend
  dependencies in the binary.
- Keep it simple, we want to deploy a single binary with no external dependencies (including DB). Code should be
  simple and easy. A bad structure is better than no structure, but a good structure is better than a bad one.

## Features

- [X] recurring chores
- [ ] recurring chores with a specific weekday (or range) (plan food for the week on weekdends)
- [X] snoozing of chores (what's the difference between snoozing and just leaving overdue? snoozing is a way to say
  "I'm not going to do this today, but I will do it tomorrow")
- [ ] one off chores (when completed they are shown as completed on the day of completion then hidden, so it's possible
  to uncomplete them)
    - keep them crossed of during the completion day then remove them

## Multitenancy

### User

- create page -> edit page (GET POST)
- edit page (GET POST)
- link login method
    - username/password (GET POST)

### Chore list

- create page (GET POST) -> view page
- view page (GET) (list of chores)
- edit page (GET POST) -> view page
- create invite page (GET POST)

# TODO

- back to main page
- remove list
- edit list

# New todo

- [ ] each link on pages will include a `?prev=` query param that will include the current page path
- [ ] each submit/post link will include a `?next=` query param that will include the next page path (normally the value
  of the prev query param)