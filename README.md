# html-to-email
This command line converts .html file to .eml file.

For embed remote images into .eml file there is [embed-email](https://github.com/gonejack/embed-email)

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/gonejack/html-to-email)
![Build](https://github.com/gonejack/html-to-email/actions/workflows/go.yml/badge.svg)
[![GitHub license](https://img.shields.io/github/license/gonejack/html-to-email.svg?color=blue)](LICENSE)

### Install
```shell
> go get github.com/gonejack/html-to-email
```

### Usage
```shell
> html-to-email *.html
```
```shell
> html-to-email -f sender@xx.com -t receiver@xx.com *.html
```
```
Flags:
  -h, --help           Show context-sensitive help.
  -f, --from=STRING    Set From field.
  -t, --to=STRING      Set To field.
  -v, --verbose        Verbose printing.
      --about          About.
```
