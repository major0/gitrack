# Themes

A theme decides how the generated site looks. The pages themselves hold only
content and structure: headings, lists, tables, links, and a few attributes a
stylesheet can aim at. Colors, fonts, spacing, separators between items, arrows
on back links, and every other bit of decoration come from the theme. Swap the
theme and every page looks different, while the HTML of every page stays exactly
the same.

## Choosing a theme

Put the theme's name in `.gitrack/config/site/theme`:

```sh
echo dark > .gitrack/config/site/theme
git track web --out ./site
```

Two themes are built in:

| name | look |
|---|---|
| `default` | light, system fonts |
| `dark` | the same layout with a dark palette |

With no file, or an empty one, the site uses `default`. A name that matches no
theme falls back to `default` rather than failing, so a typo never stops a build.
The `dark` theme pairs well with a dark code style, because the code colors are
chosen separately: put `monokai` in `.gitrack/config/site/highlight`.

## Writing your own

A theme is a directory under `.gitrack/config/site/themes/`:

```
.gitrack/config/site/themes/mine/
├── theme.css        required: the stylesheet
├── theme.js         optional: a script loaded at the end of every page
└── fonts/           optional: anything the stylesheet refers to
    └── body.woff2
```

Select it with `echo mine > .gitrack/config/site/theme`. A directory with the
same name as a built-in theme replaces that theme. A theme directory with no
`theme.css` is an error, and the message names the missing file.

Everything in the directory is copied to `assets/theme/` in the site, keeping its
layout, so a stylesheet refers to its own files by a relative path:

```css
@font-face { font-family: Body; src: url("fonts/body.woff2"); }
body { font-family: Body, serif; }
```

A good way to start is to copy a generated site's `assets/theme/theme.css` into a
new theme directory and change it.

Limits keep a stray large folder from being copied into every site: at most 200
files and 8 MiB in all. Symbolic links are skipped, so a theme cannot pull in
files from elsewhere on your machine.

## What a stylesheet can aim at

Every page has the same frame:

```
<body data-section="docs">
  <header><nav class="site-nav">  a list of links
  <main class="site-main">        the page
  <footer>
```

- `body[data-section]` names the section the page belongs to: `issues`, `portal`,
  `blog`, `docs`, `releases`, or `source`.
- Each navigation entry carries `data-section` too, and the entry for the page you
  are on has `aria-current="page"`. So the current section can be marked with
  `body[data-section="docs"] .site-nav [data-section="docs"] a`.
- Values are attributes, not just text: `data-state` (`open`, `closed`, `ready`,
  `blocked`, `released`, `upcoming`) marks tracks, issue rows, the board's lists
  and releases; `data-status` and `data-kind` appear on tracks and issue rows.

Inside the page, a class names what something is, never how it should look:

| class | on |
|---|---|
| `meta` | a short list of facts about the item above it (kind, status, author, date) |
| `count` | a number beside a heading, such as the size of the Open list |
| `empty` | the placeholder row of a list or table with nothing in it |
| `note` | a side remark, such as why a file was not highlighted |
| `sha` | an abbreviated commit id |
| `back` | the link back to a list from an item |
| `crumbs` | the breadcrumb trail on source pages |
| `title`, `status`, `waiting` | parts of a row on the issues and board pages |
| `description`, `comment`, `comment-body`, `replies` | a track's text and threaded discussion |
| `docs-layout`, `docs-nav`, `docs-content`, `toc` | the docs page frame and its lists |

Dates are `<time datetime="...">`.

Decoration is yours to add. The default theme puts a dot between items of a
`meta` list, an arrow before a `back` link, brackets around a board status, and
parentheses around a `count`, all with `::before` and `::after`. A theme that wants
none of that simply leaves them out. Read `assets/theme/theme.css` in a generated
site to see every rule.

### Scripts

`theme.js`, if present, is loaded on every page as a plain script at the end of
the body. It runs from `file://` as well as from a host. Keep in mind the page is
usable without it: a theme script should add to the page, not be needed to read
it.

### Other files a site gets

Besides the theme, a site may have `assets/highlight.css` (the code colors, only
when a page has colored code), `assets/gitrack.js` (the copy button) and
`assets/board.js` (the board's graph). They are separate from the theme so
changing a theme never changes how code is colored or what the buttons do.
