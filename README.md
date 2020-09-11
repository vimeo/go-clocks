
[![PkgGoDev](https://pkg.go.dev/badge/github.com/vimeo/go-clocks)](https://pkg.go.dev/github.com/vimeo/go-clocks)
![Go Actions](https://github.com/vimeo/go-clocks/workflows/Go/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/vimeo/go-clocks)](https://goreportcard.com/report/github.com/vimeo/go-clocks)

## go-clocks provides a modern context-aware clock interface

When writing code that involves time, it is not uncommon to want control over
the flow of that resource (particularly for tests). Many applications end up
storing a field called `now` and of type `func() time.Time`, to mock out
[`time.Now()`], but that only covers a fraction of the [`time`] package's
interface, and only covers the most trivial cases.

The [`Clock`] interface in `go-clocks` covers both querying and waiting for
time. Querying is handled via [`Clock.Now()`] and [`Clock.Until()`] while
waiting is handled by [`Clock.SleepFor()`] and [`Clock.SleepUntil()`].


### Using [`Clock`]

Generally, it is recommended store a field of type [`Clock`] in struct-types
that might need to interact with time. In production, one should use
[`DefaultClock()`], while in tests, one will generally want some combination of
[`fake.Clock`] and [`offset.Clock`].

### [`fake.Clock`] and tests

While [`Clock`] is a nice interface with both context-aware [`Clock.SleepFor()`]
and [`Clock.SleepUntil()`], it really shines when combined with the
[`fake.Clock`] in this repository's [`fake`] subpackage.

[`fake.Clock`] has a number of methods for doing implicit synchronization based
on the number of instances of [`SleepFor`][`fake.SleepFor()`]/[`SleepUntil`][`fake.SleepUntil()`] waiting on a particular
[`fake.Clock`] instance.

### [`offset.Clock`] and tests

A significantly simpler [`Clock`] implementation lives in the [`offset`]
subpackage in the form of the [`offset.Clock`] type, which adds a constant
offset to a wrapped [`Clock`]. This is useful for simulating clock-skew between
different components within a test without having to do extra bookkeeping
associated with multiple [`fake.Clock`]s.

## Dependencies

`go-clocks` very deliberately has zero dependencies aside from the Go standard
library.

## Test coverage

As of the `v1.0.0` release of go-clocks, the package has 100% test
statement-based coverage as measured by `go test -cover` in all three packages.
We endeavor to keep it that way.

[`time`]: https://golang.org/pkg/time
[`time.Now()`]: https://golang.org/pkg/time#Now
[`DefaultClock()`]: https://pkg.go.dev/github.com/vimeo/go-clocks?tab=doc#DefaultClock
[`Clock`]: https://pkg.go.dev/github.com/vimeo/go-clocks?tab=doc#Clock
[`Clock.Now()`]: https://pkg.go.dev/github.com/vimeo/go-clocks?tab=doc#Clock.Now
[`Clock.Until()`]: https://pkg.go.dev/github.com/vimeo/go-clocks?tab=doc#Clock.Until
[`Clock.SleepFor()`]: https://pkg.go.dev/github.com/vimeo/go-clocks?tab=doc#Clock.SleepFor
[`Clock.SleepUntil()`]: https://pkg.go.dev/github.com/vimeo/go-clocks?tab=doc#Clock.SleepUntil
[`fake`]: https://pkg.go.dev/github.com/vimeo/go-clocks/fake?tab=doc
[`fake.Clock`]: https://pkg.go.dev/github.com/vimeo/go-clocks/fake?tab=doc#Clock
[`fake.SleepUntil()`]: https://pkg.go.dev/github.com/vimeo/go-clocks/fake?tab=doc#Clock.SleepUntil
[`fake.SleepFor()`]: https://pkg.go.dev/github.com/vimeo/go-clocks/fake?tab=doc#Clock.SleepFor
[`offset`]: https://pkg.go.dev/github.com/vimeo/go-clocks/offset?tab=doc
[`offset.Clock`]: https://pkg.go.dev/github.com/vimeo/go-clocks/offset?tab=doc#Clock
