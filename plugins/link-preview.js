(function () {
  const cache = {};
  let tip = null;

  function internalSlug(href) {
    try {
      const url = new URL(href, location.origin);
      if (url.origin !== location.origin) return null;
      const p = url.pathname.replace(/^\/|\/$/g, '');
      if (!p || p.startsWith('api/') || p.startsWith('static/') || p === 'feed.xml') return null;
      return p;
    } catch { return null; }
  }

  function externalHost(href) {
    try { return new URL(href).hostname; } catch { return null; }
  }

  function show(anchor, html) {
    hide();
    const root = document.querySelector('.post-body') || document.body;
    tip = document.createElement('pre');
    tip.style.cssText = 'position:fixed;z-index:9000;max-width:20rem;white-space:normal;margin:0;pointer-events:none;';
    tip.innerHTML = html;
    root.appendChild(tip);
    reposition(anchor);
  }

  function reposition(anchor) {
    if (!tip) return;
    const r = anchor.getBoundingClientRect();
    const spaceBelow = window.innerHeight - r.bottom;
    tip.style.left = Math.min(r.left, window.innerWidth - tip.offsetWidth - 8) + 'px';
    if (spaceBelow > tip.offsetHeight + 8 || spaceBelow > window.innerHeight / 2) {
      tip.style.top = (r.bottom + 6) + 'px';
      tip.style.bottom = '';
    } else {
      tip.style.bottom = (window.innerHeight - r.top + 6) + 'px';
      tip.style.top = '';
    }
  }

  function hide() {
    if (tip) { tip.remove(); tip = null; }
  }

  async function fetchPost(slug) {
    if (slug in cache) return cache[slug];
    try {
      const res = await fetch('/api/v1/post/' + slug);
      cache[slug] = res.ok ? await res.json() : null;
    } catch { cache[slug] = null; }
    return cache[slug];
  }

  function esc(s) {
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
  }

  function attach(a) {
    const href = (a.getAttribute('href') || '').trim();
    if (!href || href.startsWith('#')) return;

    const slug = internalSlug(href);

    if (slug) {
      a.addEventListener('mouseenter', async () => {
        const post = await fetchPost(slug);
        if (!post) return;
        const date = post.date ? ' &middot; <em>' + esc(post.date.slice(0, 10)) + '</em>' : '';
        const desc = post.description ? '\n' + esc(post.description) : '';
        show(a, '<strong>' + esc(post.title || slug) + '</strong>' + date + desc);
      });
    } else {
      const host = externalHost(href);
      if (!host) return;
      a.addEventListener('mouseenter', () => show(a, '<code>' + esc(host) + '</code>'));
    }

    a.addEventListener('mouseleave', hide);
  }

  function init() {
    document.querySelectorAll('a[href]').forEach(attach);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
