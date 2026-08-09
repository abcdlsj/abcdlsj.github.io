<?xml version="1.0" encoding="utf-8"?>
<!--
  Pretty Feed, adapted to the abcdlsj.github.io design system.

  The feed preview reuses /static/style.css and mirrors the blog's current
  header (brand + nav + search), post list, footer and back-to-top button.
-->
<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="html" version="1.0" encoding="UTF-8" indent="yes"/>

  <!-- Format RFC1123 pubDate (e.g. "Sun, 01 Feb 2026 22:00:49 +0800") to YYYY-MM-DD -->
  <xsl:template name="fmt-date">
    <xsl:param name="d" select="''"/>
    <xsl:variable name="year" select="substring($d, 13, 4)"/>
    <xsl:variable name="month" select="substring($d, 9, 3)"/>
    <xsl:variable name="day" select="substring($d, 6, 2)"/>
    <xsl:variable name="m">
      <xsl:choose>
        <xsl:when test="$month = 'Jan'">01</xsl:when>
        <xsl:when test="$month = 'Feb'">02</xsl:when>
        <xsl:when test="$month = 'Mar'">03</xsl:when>
        <xsl:when test="$month = 'Apr'">04</xsl:when>
        <xsl:when test="$month = 'May'">05</xsl:when>
        <xsl:when test="$month = 'Jun'">06</xsl:when>
        <xsl:when test="$month = 'Jul'">07</xsl:when>
        <xsl:when test="$month = 'Aug'">08</xsl:when>
        <xsl:when test="$month = 'Sep'">09</xsl:when>
        <xsl:when test="$month = 'Oct'">10</xsl:when>
        <xsl:when test="$month = 'Nov'">11</xsl:when>
        <xsl:when test="$month = 'Dec'">12</xsl:when>
        <xsl:otherwise>00</xsl:otherwise>
      </xsl:choose>
    </xsl:variable>
    <xsl:value-of select="concat($year, '-', $m, '-', $day)"/>
  </xsl:template>

  <xsl:template match="/">
    <html xmlns="http://www.w3.org/1999/xhtml" lang="en">
      <head>
        <title><xsl:value-of select="/rss/channel/title"/> · Web Feed</title>
        <meta charset="utf-8"/>
        <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=5.0"/>
        <meta name="color-scheme" content="light dark"/>
        <link rel="stylesheet" href="/static/style.css"/>
        <style type="text/css">
          /* Feed preview specific bits; everything else comes from style.css */
          .feed-note {
            display: flex;
            align-items: flex-start;
            gap: 12px;
            margin-bottom: var(--space-8);
            padding: 12px 16px;
            background: var(--color-accent-subtle);
            border: 1px solid var(--color-accent-light);
            border-left: 3px solid var(--color-accent);
            border-radius: var(--radius-md);
            color: var(--color-text-secondary);
            font-size: 13.5px;
            line-height: 1.7;
          }
          .feed-note__icon {
            flex-shrink: 0;
            margin-top: 3px;
            color: var(--color-accent);
          }
          .feed-note strong {
            color: var(--color-text);
            font-weight: 600;
          }
          .feed-note code {
            font-family: var(--font-mono);
            font-size: 12px;
            padding: 1px 6px;
            background: var(--color-bg-elevated);
            border: 1px solid var(--color-border);
            border-radius: var(--radius-sm);
          }
          .feed-note a {
            color: var(--color-accent);
            text-decoration: underline;
            text-decoration-thickness: 1px;
            text-underline-offset: 2px;
          }
          .feed-note a:hover {
            color: var(--color-accent-hover);
          }
          .feed-link {
            display: inline-block;
            margin-top: 2px;
            font-family: var(--font-mono);
            font-size: 12px;
            color: var(--color-text-tertiary);
          }
          @media (max-width: 768px) {
            .feed-note {
              margin-bottom: var(--space-6);
            }
          }
        </style>
        <script defer="defer" src="/static/script/search.js"></script>
      </head>
      <body>
        <div class="app">
          <header class="site-header">
            <div class="site-header__inner">
              <div class="site-header__brand">
                <span class="brand-mark" aria-hidden="true"></span>
                <a href="/" class="site-header__title"><xsl:value-of select="/rss/channel/title"/></a>
                <span class="site-header__tagline"><xsl:value-of select="/rss/channel/description"/></span>
              </div>
              <nav class="site-header__nav">
                <ul class="nav-links">
                  <li><a href="/" class="nav-link">Home</a></li>
                  <li><a href="/posts" class="nav-link">Posts</a></li>
                  <li><a href="/about" class="nav-link">About</a></li>
                  <li><a href="/rss.xml" class="nav-link">Feed</a></li>
                </ul>
                <div class="site-search" id="site-search">
                  <button class="site-search__toggle" id="search-toggle" type="button" aria-label="Search posts" aria-controls="site-search-panel" aria-expanded="false">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                      <circle cx="11" cy="11" r="7"></circle>
                      <path d="m20 20-4-4"></path>
                    </svg>
                  </button>
                  <div class="site-search__panel" id="site-search-panel">
                    <label class="sr-only" for="search-input">Search posts</label>
                    <div class="site-search__field">
                      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                        <circle cx="11" cy="11" r="7"></circle>
                        <path d="m20 20-4-4"></path>
                      </svg>
                      <input id="search-input" type="search" placeholder="Search posts..." autocomplete="off" spellcheck="false"/>
                    </div>
                    <ul id="search-results" aria-label="Search results"></ul>
                    <p class="site-search__hint">Type at least 2 characters to search.</p>
                  </div>
                </div>
              </nav>
            </div>
          </header>

          <main class="container">
            <div class="feed-note">
              <svg class="feed-note__icon" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round">
                <path d="M4 11a9 9 0 0 1 9 9"></path>
                <path d="M4 4a16 16 0 0 1 16 16"></path>
                <circle cx="5" cy="19" r="2" fill="currentColor" stroke="none"></circle>
              </svg>
              <div>
                <p><strong>This is a web feed.</strong> Subscribe by copying the URL from your address bar into your newsreader.</p>
                <span class="feed-link"><xsl:text>/rss.xml</xsl:text></span>
              </div>
            </div>

            <section class="home-section">
              <div class="section-header">
                <h2 class="section-title">Recent Posts · <xsl:value-of select="count(/rss/channel/item)"/></h2>
                <a href="/posts" class="section-link">View all →</a>
              </div>

              <div class="post-list">
                <xsl:for-each select="/rss/channel/item">
                  <article class="post-item post-item--no-cover">
                    <div class="post-item__main">
                      <a class="post-item__title" target="_blank">
                        <xsl:attribute name="href"><xsl:value-of select="link"/></xsl:attribute>
                        <xsl:value-of select="title"/>
                      </a>
                      <xsl:if test="description != title and string(description) != ''">
                        <p class="post-item__excerpt"><xsl:value-of select="description"/></p>
                      </xsl:if>
                      <time class="post-item__date">
                        <xsl:attribute name="datetime"><xsl:value-of select="pubDate"/></xsl:attribute>
                        <xsl:call-template name="fmt-date">
                          <xsl:with-param name="d" select="pubDate"/>
                        </xsl:call-template>
                      </time>
                    </div>
                  </article>
                </xsl:for-each>
              </div>
            </section>
          </main>

          <footer class="footer">
            <div class="footer__inner">
              <p>© 2024 <a href="https://github.com/abcdlsj">abcdlsj</a></p>
              <p><a href="https://github.com/abcdlsj/abcdlsj.github.io">Source</a></p>
            </div>
          </footer>
        </div>

        <button class="back-to-top" type="button" aria-label="Back to top" onclick="scrollToTop()">
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="18 15 12 9 6 15"></polyline>
          </svg>
        </button>

        <script>
          function scrollToTop() {
            window.scrollTo({ top: 0, behavior: "smooth" });
          }

          const backToTop = document.querySelector(".back-to-top");
          const toggleButton = () => {
            if (!backToTop) return;
            backToTop.classList.toggle("back-to-top--visible", window.scrollY > 400);
          };
          window.addEventListener("scroll", toggleButton);
        </script>
      </body>
    </html>
  </xsl:template>
</xsl:stylesheet>
