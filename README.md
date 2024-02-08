# Reveal

A simple command line tool to quickly turn markdown into reveal.js
presentations.

## Installing

```
go install github.com/madss/reveal
```

## Usage

Create a presentation in markdown, like

```
# Title
---
## First slide
---
## Second slide
```

and then run

```
reveal presentation.md
```

That's it. See `reveal -help` for additional tweaking.