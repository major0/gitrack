'use strict';
// Drive the code-block copy button in a REAL browser (headless Chrome) and check
// what it does (ADR-0047). Run through tests/copy-button.sh, which builds the sites.
//
// Usage: node tests/copy-button.js <site-on-dir> <site-off-dir> <chrome-binary>
//
// For each clipboard situation the page is copied to a harness file with two
// scripts injected: a prelude (before the page's own scripts) that replaces the
// clipboard so every copy attempt is recorded, and a driver (after them) that
// clicks every button and writes what happened into the page. Chrome then runs the
// page and dumps the resulting DOM, which is parsed here.
//
// What this proves: the script runs in a real engine, builds the buttons on the
// right blocks only, passes the right text to the clipboard, takes the fallback
// path when the clipboard API is missing or refuses, and reports the outcome.
// What it cannot prove: that the operating system's clipboard receives the text
// (a harness clicking programmatically has no user gesture, so the recorder stands
// in for the clipboard).

const fs = require('fs');
const os = require('os');
const path = require('path');
const { spawnSync } = require('child_process');

const [onDir, offDir, chrome] = process.argv.slice(2);
if (!onDir || !offDir || !chrome) {
  console.error('usage: node copy-button.js <site-on-dir> <site-off-dir> <chrome-binary>');
  process.exit(2);
}
// os.tmpdir() honors TMPDIR; the harness pages are removed on exit.
const scratch = fs.mkdtempSync(path.join(os.tmpdir(), 'gitrack-copy-e2e-'));
process.on('exit', function () { fs.rmSync(scratch, { recursive: true, force: true }); });

let failures = 0;
function check(ok, what, detail) {
  console.log(`${ok ? 'ok  ' : 'FAIL'} ${what}${ok ? '' : '  -> ' + detail}`);
  if (!ok) failures++;
}

// The recorder. mode: 'api' (writeText works), 'reject' (writeText refuses),
// 'none' (no clipboard API at all).
function prelude(mode) {
  return `<script>
window.__copied = [];
(function () {
  var mode = ${JSON.stringify(mode)};
  if (mode === 'none') {
    Object.defineProperty(navigator, 'clipboard', { value: undefined, configurable: true });
  } else {
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: {
      writeText: function (t) {
        if (mode === 'reject') { window.__copied.push({ via: 'api-rejected', text: t }); return Promise.reject(new Error('denied')); }
        window.__copied.push({ via: 'api', text: t });
        return Promise.resolve();
      }
    }});
  }
  document.execCommand = function (cmd) {
    var ta = document.querySelector('textarea');
    window.__copied.push({ via: 'legacy', cmd: cmd, text: ta ? ta.value : null });
    return true;
  };
})();
</script>`;
}

const driver = `<script>
(function () {
  var out = { count: 0, steps: [], structure: {} };
  var btns = Array.prototype.slice.call(document.querySelectorAll('.copy-btn'));
  out.count = btns.length;
  var pres = Array.prototype.slice.call(document.querySelectorAll('pre'));
  out.structure.pre = pres.length;
  out.structure.wrapped = pres.filter(function (p) { return p.parentNode.classList.contains('code-wrap'); }).length;
  out.structure.unwrappedClasses = pres.filter(function (p) { return !p.parentNode.classList.contains('code-wrap'); })
      .map(function (p) { return p.className || '(none)'; });
  out.structure.buttonAfterPre = btns.every(function (b) { return b.previousElementSibling && b.previousElementSibling.tagName === 'PRE'; });
  out.structure.buttonNotInsidePre = btns.every(function (b) { return !b.closest('pre'); });
  out.structure.live = document.querySelectorAll('[role=status]').length;
  out.structure.ariaLabels = btns.map(function (b) { return b.getAttribute('aria-label'); });
  out.structure.type = btns.map(function (b) { return b.type; });

  var chain = Promise.resolve();
  btns.forEach(function (b, i) {
    chain = chain.then(function () {
      window.__copied.length = 0;
      b.click();
      return new Promise(function (r) { setTimeout(r, 100); }).then(function () {
        out.steps.push({ i: i, label: b.textContent, live: document.querySelector('[role=status]').textContent,
                         copied: window.__copied.slice(), activeIsButton: document.activeElement === b,
                         strayTextarea: document.querySelectorAll('textarea').length });
      });
    });
  });
  chain.then(function () {
    setTimeout(function () {
      out.afterReset = btns.map(function (b) { return b.textContent; });
      out.liveAfterReset = btns.length ? document.querySelector('[role=status]').textContent : null;
      var pre = document.createElement('pre');
      pre.id = '__result';
      pre.textContent = JSON.stringify(out);
      document.body.appendChild(pre);
    }, 2500);
  });
})();
</script>`;

function unescapeHTML(s) { return s.replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&quot;/g, '"').replace(/&amp;/g, '&'); }

function runPage(siteDir, rel, mode, transform) {
  let page = fs.readFileSync(path.join(siteDir, rel), 'utf8');
  if (transform) { page = transform(page); }
  const harness = page.replace('<head>', '<head>' + prelude(mode)).replace(/<\/body>/, driver + '</body>');
  // The harness page sits beside the real one, so the relative links to assets/
  // resolve exactly as they do on the site; it is removed once Chrome has run.
  const file = path.join(siteDir, path.dirname(rel), '__harness-' + mode + (transform ? '-mixed' : '') + '.html');
  fs.writeFileSync(file, harness);
  process.on('exit', function () { fs.rmSync(file, { force: true }); });
  const r = spawnSync(chrome, ['--headless=new', '--no-sandbox', '--disable-gpu', '--virtual-time-budget=9000',
    '--dump-dom', 'file://' + file], { encoding: 'utf8', timeout: 90000 });
  if (r.status !== 0) { return { error: 'chrome exited ' + r.status + ': ' + (r.stderr || '').slice(0, 300) }; }
  const m = r.stdout.match(/<pre id="__result">([\s\S]*?)<\/pre>/);
  if (!m) { return { error: 'no result element in the dumped DOM', dom: r.stdout.slice(0, 400) }; }
  return JSON.parse(unescapeHTML(m[1]));
}

const guideExpect = ['go install example.com/x@latest\necho "a <b> & c"\n', 'plain block\n  with indentation\n'];
const blobExpect = 'package main\n\nfunc main() {\n\tprintln("<hi> & bye")\n}\n';

console.log('== docs page, clipboard API works ==');
let g = runPage(onDir, 'docs/guide.html', 'api');
if (g.error) { check(false, 'ran', g.error + ' ' + (g.dom || '')); } else {
  check(g.count === 2, 'two buttons, one per code block', 'got ' + g.count);
  check(g.structure.wrapped === 2 && g.structure.pre === 2, 'both blocks wrapped', JSON.stringify(g.structure));
  check(g.structure.buttonAfterPre && g.structure.buttonNotInsidePre, 'each button sits beside its block, not inside it', JSON.stringify(g.structure));
  check(g.structure.live === 1, 'one live region', 'got ' + g.structure.live);
  check(g.structure.ariaLabels.every(function (l) { return l === 'Copy code to clipboard'; }), 'buttons have an accessible name', JSON.stringify(g.structure.ariaLabels));
  check(g.structure.type.every(function (t) { return t === 'button'; }), 'buttons are type=button (never submit)', JSON.stringify(g.structure.type));
  g.steps.forEach(function (s, i) {
    check(s.copied.length === 1 && s.copied[0].via === 'api', 'block ' + i + ': copied once via the clipboard API', JSON.stringify(s.copied));
    check(s.copied[0] && s.copied[0].text === guideExpect[i], 'block ' + i + ': exact code text (no button label, escaped chars intact)',
      JSON.stringify(s.copied[0] && s.copied[0].text));
    check(s.label === 'Copied' && s.live === 'Code copied to clipboard', 'block ' + i + ': label and live region report success', JSON.stringify([s.label, s.live]));
    check(s.activeIsButton, 'block ' + i + ': focus stays on the button', 'focus moved');
  });
  check(g.afterReset.every(function (l) { return l === 'Copy'; }), 'labels reset to "Copy"', JSON.stringify(g.afterReset));
  check(g.liveAfterReset === '', 'live region cleared', JSON.stringify(g.liveAfterReset));
}

console.log('== docs page, clipboard API refuses -> falls back ==');
g = runPage(onDir, 'docs/guide.html', 'reject');
if (g.error) { check(false, 'ran', g.error); } else {
  const s = g.steps[0];
  check(s.copied.length === 2 && s.copied[0].via === 'api-rejected' && s.copied[1].via === 'legacy', 'tried the API, then the fallback', JSON.stringify(s.copied));
  check(s.copied[1] && s.copied[1].text === guideExpect[0], 'fallback copies the same exact text', JSON.stringify(s.copied[1]));
  check(s.label === 'Copied', 'reports success after the fallback', s.label);
  check(s.strayTextarea === 0, 'the temporary textarea is removed', 'left ' + s.strayTextarea);
}

console.log('== docs page, no clipboard API at all ==');
g = runPage(onDir, 'docs/guide.html', 'none');
if (g.error) { check(false, 'ran', g.error); } else {
  const s = g.steps[0];
  check(s.copied.length === 1 && s.copied[0].via === 'legacy' && s.copied[0].text === guideExpect[0], 'uses the fallback directly', JSON.stringify(s.copied));
  check(s.label === 'Copied', 'reports success', s.label);
}

// The Go side only attaches the script to pages that hold a code block, so on the
// real site no page mixes code blocks with other <pre> elements. The script still
// has to pick only code blocks, in case a future page does, so mix them in here.
console.log('== the script itself picks only code blocks (mixed page) ==');
const mixed = runPage(onDir, 'docs/guide.html', 'api', function (html) {
  return html.replace('<footer',
    '<pre class="mermaid">graph LR</pre><pre class="diff"><span>+x</span></pre>' +
    '<pre>bare description text</pre><pre><span>first child is not code</span></pre><footer');
});
if (mixed.error) { check(false, 'ran', mixed.error); } else {
  check(mixed.count === 2, 'still only the two real code blocks get a button', 'got ' + mixed.count);
  check(mixed.structure.pre === 6, 'six <pre> on the page (2 code + 4 injected)', 'got ' + mixed.structure.pre);
  const left = mixed.structure.unwrappedClasses.slice().sort().join(',');
  check(left === '(none),(none),diff,mermaid', 'the four others are left unwrapped', left);
}

console.log('== source page: numbered lines are not part of the copied text ==');
let b = runPage(onDir, 'source/blob/main.go/index.html', 'api');
if (b.error) { check(false, 'ran', b.error); } else {
  check(b.count === 1, 'one button', 'got ' + b.count);
  check(b.steps[0] && b.steps[0].copied[0] && b.steps[0].copied[0].text === blobExpect, 'copies exactly the file, tabs and all, with no line numbers',
    JSON.stringify(b.steps[0] && b.steps[0].copied[0] && b.steps[0].copied[0].text));
}

console.log('== pages that must NOT get buttons ==');
for (const rel of ['board.html', 'index.html', 'issues.html', 'source/index.html', 'source/log.html']) {
  const r = runPage(onDir, rel, 'api');
  if (r.error) { check(false, rel + ' ran', r.error); continue; }
  check(r.count === 0, rel + ': no buttons (' + r.structure.pre + ' <pre> left alone: ' + JSON.stringify(r.structure.unwrappedClasses) + ')', 'got ' + r.count);
}
const trackFile = fs.readdirSync(path.join(onDir, 'track')).find(function (f) { return f.endsWith('.html'); });
const tr = runPage(onDir, 'track/' + trackFile, 'api');
if (tr.error) { check(false, 'track page ran', tr.error); } else {
  check(tr.count === 0 && tr.structure.pre >= 1, 'track page: its description <pre> has no button (' + tr.structure.pre + ' <pre>)', JSON.stringify(tr));
}

console.log('== with the button turned off ==');
for (const rel of ['docs/guide.html', 'source/blob/main.go/index.html']) {
  const html = fs.readFileSync(path.join(offDir, rel), 'utf8');
  check(!html.includes('<script') && !html.includes('gitrack.js'), rel + ': no script element and no link to the script', 'script present');
  const r = runPage(offDir, rel, 'api');
  check(!r.error && r.count === 0, rel + ': no buttons in a real browser', JSON.stringify(r).slice(0, 200));
}

console.log(failures === 0 ? 'ALL_OK' : failures + ' FAILED');
process.exit(failures === 0 ? 0 : 1);
