const browserType = process.argv[2] || 'chromium';
const { [browserType]: pw } = require('playwright');

(async () => {
  const browser = await pw.launch({ headless: true });
  const page = await browser.newPage({ colorScheme: process.env.SCHEME || 'light' });
  page.on('console', (m) => console.log('console:', m.text()));
  page.on('pageerror', (e) => console.log('pageerror:', e.message));

  // Sandbox can't reach Google Fonts; strip the @import so style.css applies.
  await page.route('**/static/style.css', async (route) => {
    const resp = await route.fetch();
    let body = await resp.text();
    body = body.replace(/@import[^;]+;/g, '');
    await route.fulfill({ response: resp, body });
  });

  await page.goto('http://127.0.0.1:3001/rss.xml', {
    waitUntil: 'commit',
    timeout: 15000,
  });
  await page.waitForTimeout(4000);

  const before = await page.evaluate(() => {
    const searchToggle = document.querySelector('.site-search__toggle');
    const oldToggle = document.querySelector('.theme-toggle');
    const searchRoot = document.querySelector('#site-search');
    const body = document.querySelector('body');
    const root = document.documentElement;
    return {
      title: document.title,
      readyState: document.readyState,
      allElements: document.querySelectorAll('*').length,
      htmlLength: root.outerHTML.length,
      dataTheme: root.getAttribute('data-theme'),
      rootTag: root.tagName,
      bodyExists: !!body,
      bodyChildren: body ? body.children.length : null,
      bodyBg: body ? getComputedStyle(body).backgroundColor : null,
      oldThemeToggleExists: !!oldToggle,
      searchToggleExists: !!searchToggle,
      searchToggleClass: searchToggle ? searchToggle.className : null,
      searchToggleLabel: searchToggle ? searchToggle.ariaLabel : null,
      searchToggleDisplay: searchToggle ? getComputedStyle(searchToggle).display : null,
      searchToggleWidth: searchToggle ? getComputedStyle(searchToggle).width : null,
      searchToggleRect: searchToggle ? JSON.parse(JSON.stringify(searchToggle.getBoundingClientRect())) : null,
      searchRootExists: !!searchRoot,
      headerSearchIcons: document.querySelectorAll('.site-search__toggle svg, .site-search__field svg').length,
      scriptCount: document.scripts.length,
      styleSheets: document.styleSheets.length,
      sheetHref: document.styleSheets[0] ? document.styleSheets[0].href : null,
      sheetRules: (() => {
        try {
          const s = document.styleSheets[0];
          return s ? s.cssRules.length : -1;
        } catch (e) {
          return 'err:' + e.message;
        }
      })(),
      scripts: Array.from(document.scripts).map((s) => s.textContent.slice(0, 200)),
      resources: performance.getEntriesByType('resource').map((r) => r.name),
      firstText: body ? body.innerText.slice(0, 120) : null,
    };
  });
  console.log('BEFORE:', JSON.stringify(before, null, 2));

  const clickInfo = await page.evaluate(async () => {
    const root = document.querySelector('#site-search');
    const toggle = document.querySelector('#search-toggle');
    let openBefore = null;
    let openAfter = null;
    if (root && toggle) {
      openBefore = root.classList.contains('site-search--open');
      toggle.click();
      openAfter = root.classList.contains('site-search--open');
    }
    const input = document.querySelector('#search-input');
    let searchResultCount = null;
    if (input) {
      input.value = 'go';
      input.dispatchEvent(new Event('input', { bubbles: true }));
      await new Promise((r) => setTimeout(r, 1200));
      const results = document.querySelector('#search-results');
      searchResultCount = results ? results.children.length : -1;
    }
    return {
      hasRoot: !!root,
      hasToggle: !!toggle,
      openBefore,
      openAfter,
      searchResultCount,
    };
  });
  console.log('CLICK:', JSON.stringify(clickInfo, null, 2));

  const realClick = await page.evaluate(() => {
    const root = document.querySelector('#site-search');
    const toggle = document.querySelector('#search-toggle');
    if (root && toggle) {
      toggle.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, view: window }));
    }
    return root ? root.classList.contains('site-search--open') : null;
  });
  console.log('REAL CLICK OPEN:', realClick);

  const styles = await page.evaluate(() => {
    const pick = (sel) => {
      const el = document.querySelector(sel);
      if (!el) return null;
      const cs = getComputedStyle(el);
      return {
        display: cs.display,
        width: cs.width,
        padding: cs.padding,
        fontFamily: cs.fontFamily.slice(0, 40),
        color: cs.color,
        backgroundColor: cs.backgroundColor,
        borderBottom: cs.borderBottomWidth + ' ' + cs.borderBottomStyle + ' ' + cs.borderBottomColor,
      };
    };
    return {
      feedNote: pick('.feed-note'),
      siteHeader: pick('.site-header'),
      sectionTitle: pick('.section-title'),
      postItem: pick('.post-item'),
      footer: pick('.footer'),
      searchToggle: pick('.site-search__toggle'),
    };
  });
  console.log('STYLES:', JSON.stringify(styles, null, 2));

  const matchInfo = await page.evaluate(() => {
    const btn = document.querySelector('.theme-toggle');
    const out = {
      ns: btn ? btn.namespaceURI : null,
      tag: btn ? btn.tagName : null,
      class: btn ? btn.getAttribute('class') : null,
      matchesClass: btn ? btn.matches('.theme-toggle') : null,
      matchesButtonClass: btn ? btn.matches('button.theme-toggle') : null,
      matchesButton: btn ? btn.matches('button') : null,
    };
    out.rules = [];
    out.allSelectors = [];
    try {
      for (const sheet of document.styleSheets) {
        for (const rule of sheet.cssRules) {
          if (rule.selectorText) {
            if (rule.selectorText.toLowerCase().includes('theme') || rule.selectorText.toLowerCase().includes('toggle')) {
              out.allSelectors.push({ href: sheet.href, selector: rule.selectorText });
            }
          }
          if (rule.selectorText && rule.selectorText.includes('theme-toggle') && btn) {
            out.rules.push({
              href: sheet.href,
              selector: rule.selectorText,
              matches: btn.matches(rule.selectorText),
            });
          }
        }
      }
    } catch (e) {
      out.err = e.message;
    }
    out.externalHrefs = Array.from(document.styleSheets).map((s) => s.href);
    out.externalRuleCounts = Array.from(document.styleSheets).map((s) => {
      try {
        return s.cssRules.length;
      } catch (e) {
        return -1;
      }
    });
    return out;
  });
  console.log('MATCH:', JSON.stringify(matchInfo, null, 2));

  await page.screenshot({ path: '/tmp/rss-light.png', fullPage: true, timeout: 15000 });
  await browser.close();
})();
