# Redsync

[![Build Status](https://travis-ci.org/rgalanakis/redsync.svg?branch=master)](https://travis-ci.org/rgalanakis/redsync)
[![codecov](https://codecov.io/gh/rgalanakis/redsync/branch/master/graph/badge.svg)](https://codecov.io/gh/rgalanakis/redsync)

[![GoDoc](https://godoc.org/github.com/rgalanakis/redsync?status.svg)](http://godoc.org/github.com/rgalanakis/redsync)
[![license](http://img.shields.io/badge/license-BSDv2-orange.svg)](https://raw.githubusercontent.com/rgalanakis/redsync/master/LICENSE)

Redsync provides a Redis-based distributed mutual exclusion lock implementation for Go as described in
[this post](http://redis.io/topics/distlock).

A reference library (by [antirez](https://github.com/antirez)) for Ruby is available at
[github.com/antirez/redlock-rb](https://github.com/antirez/redlock-rb).

## Installation

Install Redsync using the go get command:

    $ go get github.com/rgalanakis/redsync

Dependencies are the Go distribution and [Redigo](https://github.com/gomodule/redigo).
It also [Redigomock](https://github.com/rafaeljusto/redigomock) for testing.

## Documentation

- [Reference](http://godoc.org/github.com/rgalanakis/redsync)

## Contributing

Contributions are welcome.

## License

Redsync is available under the [BSD (3-Clause) License](https://opensource.org/licenses/BSD-3-Clause).

## Disclaimer

This code implements an algorithm which is currently a proposal, it was not formally analyzed.
Make sure to understand how it works before using it in production environments.
