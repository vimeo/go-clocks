
[![PkgGoDev](https://pkg.go.dev/badge/github.com/vimeo/go-clocks)](https://pkg.go.dev/github.com/vimeo/go-clocks)
![Go Actions](https://github.com/vimeo/go-clocks/workflows/Go/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/vimeo/go-clocks)](https://goreportcard.com/report/github.com/vimeo/go-clocks)

## go-clocks provides a modern context-aware clock interface

When writing code that involves time, it is not uncommon to want control over
the flow of that resource. Many applications end up storing a
field called `now` and of type `func() time.Time`, to mock out time, but that
only covers a fraction of the `time` package's interface, and only covers the
most trivial cases.

The `Clock` interface in `go-clocks` covers both querying and waiting for time.

### `fake.Clock` and tests

While `Clock` is a nice interface with both context-aware `SleepFor` and
`SleepUntil`, it really shines when combined with the `Clock` in this
repository.

`fake.Clock` has a number of methods for doing implicit synchronization based on
the number of instances of `SleepFor`/`SleepUntil` waiting on a particular
`fake.Clock` instance.
