const browserType = process.argv[2] || 'chromium';
const { [browserType]: pw } = require('playwright');

(async () => {
  const browser = await pw.launch({ headless: true });
  const page = await browser.newPage({ colorScheme: process.env.SCHEME || 'light' });
  page.on('console', (m) => console.log('console:', m.text()));
  page.on('pageerror', (e) => console.log('pageerror:', e.message));

  await page.goto('http://127.0.0.1:3001/rss.xml', {
    waitUntil: 'commit',
    timeout: 15000,
  });
  await page.waitForTimeout(4000);

  const before = await page.evaluate(() => {
    const moon = document.querySelector('.icon-moon');
    const sun = document.querySelector('.icon-sun');
    const btn = document.querySelector('.theme-toggle');
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
      btnExists: !!btn,
      btnClass: btn ? btn.className : null,
      btnText: btn ? btn.ariaLabel : null,
      moonDisplay: moon ? getComputedStyle(moon).display : 'no-el',
      sunDisplay: sun ? getComputedStyle(sun).display : 'no-el',
      moonRect: moon ? JSON.parse(JSON.stringify(moon.getBoundingClientRect())) : null,
      sunRect: sun ? JSON.parse(JSON.stringify(sun.getBoundingClientRect())) : null,
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

  const clickInfo = await page.evaluate(() => {
    const btn = document.querySelector('.theme-toggle');
    let before = null;
    let after = null;
    if (btn && typeof toggleTheme === 'function') {
      before = document.documentElement.getAttribute('data-theme');
      btn.click();
      after = document.documentElement.getAttribute('data-theme');
    }
    return {
      hasBtn: !!btn,
      hasToggleTheme: typeof toggleTheme === 'function',
      before,
      after,
    };
  });
  console.log('CLICK:', JSON.stringify(clickInfo, null, 2));

  await page.screenshot({ path: '/tmp/rss-light.png', fullPage: true });
  await browser.close();
})();
