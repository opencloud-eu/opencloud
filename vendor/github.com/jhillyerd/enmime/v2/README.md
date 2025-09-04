# enmime
[![PkgGoDev](https://pkg.go.dev/badge/github.com/jhillyerd/enmime)][Pkg Docs]
[![Build and Test](https://github.com/jhillyerd/enmime/actions/workflows/build-and-test.yml/badge.svg)](https://github.com/jhillyerd/enmime/actions/workflows/build-and-test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jhillyerd/enmime)][Go Report Card]
[![Coverage Status](https://coveralls.io/repos/github/jhillyerd/enmime/badge.svg?branch=main)][Coverage Status]


enmime is a MIME encoding and decoding library for Go, focused on generating and
parsing MIME encoded emails.  It is being developed in tandem with the
[Inbucket] email service.

enmime includes a fluent interface builder for generating MIME encoded messages,
see the wiki for example [Builder Usage].

See our [Pkg Docs] for examples and API usage information.


## Development Status

enmime is production quality, but there are many buggy MIME encoders in the
wild, so you may still encounter messages it cannot parse.

See [CONTRIBUTING.md] for more information.


## About

enmime is written in [Go][Golang].

enmime is open source software released under the MIT License.  The latest
version can be found at https://github.com/jhillyerd/enmime


[Builder Usage]:   https://github.com/jhillyerd/enmime/wiki/Builder-Usage
[Coverage Status]: https://coveralls.io/github/jhillyerd/enmime
[CONTRIBUTING.md]: https://github.com/jhillyerd/enmime/blob/main/CONTRIBUTING.md
[Inbucket]:        https://www.inbucket.org/
[Golang]:          https://go.dev/
[Go Report Card]:  https://goreportcard.com/report/github.com/jhillyerd/enmime
[Pkg Docs]:        https://pkg.go.dev/github.com/jhillyerd/enmime
