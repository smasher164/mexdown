# mexdown

[![GoDoc](https://godoc.org/akhil.cc/mexdown?status.svg)](https://godoc.org/akhil.cc/mexdown)
[![Build Status](https://travis-ci.org/smasher164/mexdown.svg?branch=master)](https://travis-ci.org/smasher164/mexdown)
[Blog post](https://www.blog.akhil.cc/mexdown)

mexdown is a lightweight integrating markup language. It offers a syntax similar to markdown's, but its distinguishing feature is in integrating commands to generate a document.

These commands are processed through **directives**. Simply append a command to a block of raw text, like so:

````
```dot -Tsvg
digraph g{
  rankdir=LR;
  "A" -> "B" -> "C"
  "B" -> "D"
}
```
````

[**`dot`**](https://www.graphviz.org/doc/info/command.html) is a command installed with Graphviz that reads a graph's description into standard input, and writes its SVG representation to standard output. This SVG can be embedded into an HTML document, so the following mexdown source file

````
# Title About Graphs

The following graph illustrates my point:
```dot -Tsvg
digraph g{
  rankdir=LR;
  "A" -> "B" -> "C"
  "B" -> "D"
}
```
````

when run through mexdown's html backend either via the command line,

```
$ mexdown html graphs.xd
```

or programmatically with Go,

```
file := parser.MustParse(src)
out, err := html.Gen(file).Output()
```

produces the following HTML document

![A graph embedded in an HTML document](https://lh3.googleusercontent.com/OV8TUBGbmJnjrYWXqIa2mDb9aOK-sbBotMU_zACIees0DHi3bvgjGD7mnlnp0evREdflqgk6VBODzLq6Pd7n0cLrQnH_r-4dYZa0Bm8xs6zwAS43r2534y4V90OEyk_r3TXh0KcS2PdLNrJusfk8PGfzC8e0BjJQZT_cEXoj9Qe-8zMQOS9MsihPF9U7EN-Xrv3f2qp_dGsPygj9Zzmwjj7JxZ8nYDkPLxl9dsaHzYb4YFd8sbaEyZLyNjMo3X4g3v6Uy298TeZngoVeTLUk1DegEKNHVKOf11L4fdMOLV6EWhZSZ19Uf7-EKOUBv3o09QpuiUz87-dK0C2PjVkTeV1pyIsblnYWcbRtz6QRj8acX5x-k3_v9SVPImmIT52OYebVqYL3CFGRdDr5zmi6H-Rq0d9pL4gQC3L_ip6hS-NeAIOn9iqXektmAki7iIT1hMcmchffAWpiQagTanB7cGtjjuv3YdKk8Np3F9EmIyX3o7q8al03QNNLvTCUTgPRWmzeo6m85TWA48LF_5voWR7oe_RGvWCGXeB92HVNRJCmzxomaL00XZNC4mAUYp16pIHqi92Yvw05mQrbDdWgi_7l67W9N36wZCBN6TI=w375-h270-no "Graph Example")

Directives enable a powerful way to control document generation, without building in new language features into markdown itself. As long as the command reads from STDIN and writes to STDOUT, you can
- Embed mathematical equations.
- Write literate programs.
- Generate hyperlinked source-code documentation.
- Create a table of contents.
- Embed interactive widgets based off descriptions.
- Use it in other creative ways!

The language itself is backend-independent, and the grammar was written with this in mind. For example, the grammar doesn't dictate the formatting of list items, the escaping of raw text, or the command language used for the directives.

## Supported Backends

Currently, the only implemented backend is HTML. However, the next candidates are
- PDF
- Postscript
- Latex
- Google Docs/Slides
- Pandoc

Anyone can implement their own backend, since it only needs the AST, as defined in [akhil.cc/mexdown/ast](https://akhil.cc/mexdown/ast).

## Contributing
Please file issues on Github's issue tracker. There is still a lot of work that needs to be done before creating a release. Thank you for taking the time to contribute!