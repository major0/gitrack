# Publishing

The generated site is a plain directory with no host coupling, so publishing is
just copying files. There is no application server in the deployment path — a
smaller attack surface, less energy, and no hosted dependency.

## Build, then copy anywhere

```sh
git track web --out ./site
```

Then deploy `./site` with whatever moves files:

```sh
aws s3 sync ./site s3://my-bucket/ --delete     # S3 (static website hosting)
rsync -a --delete ./site/ user@host:/var/www/   # any web server
cp -r ./site /mnt/shared/project-site            # a shared drive
ipfs add -r ./site                               # IPFS
```

The same output works from `file://` — open `./site/index.html` directly, no
server needed. Links are relative, so the site also works from a subpath
(`https://host/projects/foo/`).

## GitHub Pages

A natural target, since the tracker already lives in the repo. A minimal
workflow rebuilds and deploys the site whenever the tracker changes:

```yaml
name: docs
on:
  push:
    paths: ['.gitrack/**', 'docs/**']
permissions:
  contents: read
  pages: write
  id-token: write
jobs:
  build-deploy:
    runs-on: ubuntu-latest
    environment:
      name: github-pages
      url: ${{ steps.deploy.outputs.page_url }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: 'stable' }
      - run: go install github.com/major0/gitrack/cmd/git-track@latest
      - run: git track web --out ./site
      - uses: actions/upload-pages-artifact@v3
        with: { path: ./site }
      - id: deploy
        uses: actions/deploy-pages@v4
```

Trigger on both `.gitrack/**` (tracker changes) and `docs/**` (documentation
changes) so the site rebuilds for either.

## Preview before you publish

Use the live server to check the site locally without writing files:

```sh
git track serve
```

It renders the current tracker on every request. When it looks right, run
`git track web --out ./site` to produce the deployable snapshot.

## Why static, decentralized publishing

gitrack exists to let a team run a tracker without a hosted service. Static
output is the other half of that story: it serves from anything, costs a
fraction of the energy of an always-on app, and has no running endpoint to
exploit, no database to inject, no auth to bypass. The tracker travels with the
repo; the site is just its rendered, portable snapshot. See
[the project vision](../VISION.md).
