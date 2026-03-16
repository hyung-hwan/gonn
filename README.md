# gonn

`gonn` is a Go translation of the original `ronn` project.
The original Ruby codebase has been translated to Golang in this project.

Based on `ronn`, `gonn` converts ronn-style Markdown documents into
manpage output formats such as roff and HTML. It keeps the same general
purpose as the original project: writing Unix manual pages in readable
Markdown and rendering them into formats that are practical for command
line documentation and web publishing.

This Go port includes both library code and a command-line interface, so
it can be used as a reusable package or as a standalone tool for building
manual pages.

## Credit

`gonn` is based on the original `ronn` project by Ryan Tomayko and the
other contributors to `ronn`. Credit for the original design, document
format, and Ruby implementation belongs to the `ronn` project.

Original project:

- https://github.com/rtomayko/ronn
