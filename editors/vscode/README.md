# goboot Annotations for VS Code

Syntax highlighting and snippets for [goboot](https://github.com/zombocoder/goboot)
annotations in Go doc comments.

goboot is a compile-time framework whose annotations live in ordinary Go
comments — `// @Service`, `// @GetMapping(path="/x")`, `` // @Query(`SELECT ...`) ``.
Plain Go tooling renders those as flat comment text; this extension colors them
so the annotation name, arguments, strings, numbers, and enums stand out.

## Features

- **Highlighting** — an injection grammar that lights up `@Annotation(args)`
  inside `//` and `/* */` comments in Go files. Names, argument keys, `=`,
  strings/raw strings, numbers, booleans, `null`, arrays/objects, and dotted
  enums (`TimeUnit.MINUTES`) each get their own scope.
- **Snippets** — type `@Service`, `@GetMapping`, `@Repository`, `@Query`,
  `@Scheduled`, `@ControllerAdvice`, … in a Go file for a ready-to-fill
  annotation.

## Install

From a packaged VSIX:

```bash
cd editors/vscode
npx @vscode/vsce package        # produces goboot-annotations-<version>.vsix
code --install-extension goboot-annotations-*.vsix
```

Then open any Go file with goboot annotations. No settings required — the
grammar is injected into `source.go` automatically.

## How it works

The grammar (`syntaxes/goboot.tmLanguage.json`) has an
`injectionSelector` of `L:comment.line.double-slash.go, L:comment.block.go`, so
it applies **only inside Go comments** and layers on top of the base Go grammar.
It matches `@` + a PascalCase name and, when present, the parenthesized argument
list.

> Note: an annotation name mentioned in prose (e.g. "the @Transactional method")
> is highlighted too — which mirrors how goboot's own parser treats any
> comment-line-leading `@Name` as an annotation.

## Scopes (for theme authors)

| Token             | Scope                                          |
| ----------------- | ---------------------------------------------- |
| `@`               | `punctuation.definition.annotation.goboot`     |
| annotation name   | `storage.type.annotation.goboot`               |
| argument key      | `variable.parameter.annotation.goboot`         |
| `=`               | `keyword.operator.assignment.goboot`           |
| string / raw      | `string.quoted.double.goboot` / `.other.raw.`  |
| number            | `constant.numeric.goboot`                      |
| `true`/`false`/`null` | `constant.language.goboot`                 |
| enum / identifier | `constant.other.enum.goboot`                   |

## Development

```bash
cd editors/vscode/test && npm install     # vscode-textmate + vscode-oniguruma
node tokenize.js                           # grammar tokenization tests
```

Licensed under Apache-2.0 (same as goboot).
