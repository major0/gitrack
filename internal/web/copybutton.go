package web

import (
	"path/filepath"
	"strings"

	"github.com/major0/gitrack/internal/core"
)

// copyConfigFile is the setting for the copy button (ADR-0047): "off" turns it
// off, anything else (or no file) leaves it on.
const copyConfigFile = siteConfigDir + "/copy"

// pageExtra is what a page needs beyond the theme: the syntax-highlighting
// stylesheet, the copy script (for pages with code blocks), or the board script.
// All are false for most pages.
type pageExtra struct {
	Highlight bool
	Code      bool
	Board     bool
}

// copyScript adds a "Copy" button to every code block on the page (ADR-0047). It
// is progressive enhancement: the buttons are created by this script, so a visitor
// without JavaScript, or a host whose content-security policy forbids inline
// scripts, sees the page exactly as before and no dead control.
//
// It makes no network request and uses no library. A block is a <pre> holding a
// <code> (a Markdown fence, highlighted or not) or a source page's numbered
// <pre class="blob">. Other <pre> elements (the board graph, diffs, commit
// messages, issue descriptions) are left alone. Each block is wrapped in a
// positioned <div> and the button sits beside it, not inside it, so copying reads
// the code's own text and never the button's. Line numbers on source pages come
// from CSS, so they are not part of that text either.
//
// Copying uses the asynchronous clipboard API where it exists, and falls back to
// selecting a hidden textarea and execCommand('copy'), which also works from
// file:// where the clipboard API may be withheld. The result is announced through
// a polite live region as well as in the button's label.
//
// The text contains no "</script>" and no URL.
const copyScript = `(function () {
  var pres = document.querySelectorAll('pre');
  var blocks = [];
  for (var i = 0; i < pres.length; i++) {
    var p = pres[i];
    var first = p.firstElementChild;
    if (p.classList.contains('blob') || (first && first.tagName === 'CODE')) {
      blocks.push(p);
    }
  }
  if (!blocks.length) { return; }

  var live = document.createElement('div');
  live.className = 'sr-only';
  live.setAttribute('role', 'status');
  document.body.appendChild(live);

  function legacy(text) {
    var ta = document.createElement('textarea');
    ta.value = text;
    ta.setAttribute('readonly', '');
    ta.style.position = 'fixed';
    ta.style.top = '0';
    ta.style.opacity = '0';
    document.body.appendChild(ta);
    ta.select();
    var ok = false;
    try { ok = document.execCommand('copy'); } catch (e) { ok = false; }
    document.body.removeChild(ta);
    return ok;
  }

  function copy(text) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      return navigator.clipboard.writeText(text).then(
        function () { return true; },
        function () { return legacy(text); }
      );
    }
    return Promise.resolve(legacy(text));
  }

  blocks.forEach(function (pre) {
    var wrap = document.createElement('div');
    wrap.className = 'code-wrap';
    pre.parentNode.insertBefore(wrap, pre);
    wrap.appendChild(pre);

    var btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'copy-btn';
    btn.textContent = 'Copy';
    btn.setAttribute('aria-label', 'Copy code to clipboard');
    var timer = null;
    btn.addEventListener('click', function () {
      copy(pre.textContent).then(function (ok) {
        btn.textContent = ok ? 'Copied' : 'Copy failed';
        live.textContent = ok ? 'Code copied to clipboard' : 'Copy failed';
        btn.focus();
        clearTimeout(timer);
        timer = setTimeout(function () {
          btn.textContent = 'Copy';
          live.textContent = '';
        }, 2000);
      });
    });
    wrap.appendChild(btn);
  });
})();`

// copyEnabled reports whether the copy button is on for this generation. It reads
// the setting once.
func (sc *siteContext) copyEnabled() bool {
	if !sc.copyLoaded {
		sc.copyLoaded = true
		v, err := readConfigValue(filepath.Join(sc.repo.Root, core.Dir, copyConfigFile), "")
		// a missing or unreadable file leaves the button on; only "off" disables it
		sc.copyOn = err != nil || !strings.EqualFold(v, "off")
	}
	return sc.copyOn
}

// extraFor builds a page's extras: whether it uses the highlight stylesheet and
// whether it has code blocks (the copy script is attached to those unless the
// button is turned off).
func extraFor(highlighted, hasCode bool) pageExtra {
	return pageExtra{Highlight: highlighted, Code: hasCode}
}
