let searchIndex = null;
const searchCache = new Map();

async function loadSearchIndex() {
  if (searchIndex !== null) return searchIndex;

  try {
    const response = await fetch('/search-index.json');
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    searchIndex = await response.json();
  } catch (error) {
    console.error('Failed to load the search index:', error);
    searchIndex = { words: {}, posts: {} };
  }

  return searchIndex;
}

async function search(query) {
  await loadSearchIndex();

  const normalizedQuery = query.trim().toLowerCase();
  if (searchCache.has(normalizedQuery)) {
    return searchCache.get(normalizedQuery);
  }

  const words = normalizedQuery.split(/\s+/).filter(Boolean);
  const results = new Map();

  for (const word of words) {
    for (const postURL of searchIndex.words[word] || []) {
      results.set(postURL, (results.get(postURL) || 0) + 1);
    }

    for (const [postURL, post] of Object.entries(searchIndex.posts || {})) {
      if ((post.title || '').toLowerCase().includes(word)) {
        results.set(postURL, (results.get(postURL) || 0) + 2);
      }
    }
  }

  const sortedResults = Array.from(results.entries())
    .sort((a, b) => b[1] - a[1])
    .map(([url]) => url);

  searchCache.set(normalizedQuery, sortedResults);
  return sortedResults;
}

function escapeHTML(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#039;');
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function highlightText(text, query) {
  const safeText = escapeHTML(text);
  const words = query.trim().split(/\s+/).filter(Boolean).map(escapeRegExp);
  if (words.length === 0) return safeText;

  return safeText.replace(new RegExp(`(${words.join('|')})`, 'gi'), '<mark>$1</mark>');
}

function formatSearchDate(value) {
  return value ? escapeHTML(value.slice(0, 10)) : '';
}

function displayResults(results, query) {
  const searchResults = document.getElementById('search-results');
  if (!searchResults) return;

  if (results.length === 0) {
    searchResults.innerHTML = '<li class="search-no-results">No matching posts.</li>';
    return;
  }

  searchResults.innerHTML = results.slice(0, 8).map((url) => {
    const post = searchIndex.posts?.[url] || {};
    const title = post.title || url;
    const date = formatSearchDate(post.date);

    return `
      <li class="search-result-item">
        <a href="/posts/${escapeHTML(url)}.html" class="search-result-link">
          <span class="search-result-title">${highlightText(title, query)}</span>
          ${date ? `<time class="search-result-date">${date}</time>` : ''}
        </a>
      </li>
    `;
  }).join('');
}

function showSearchStatus(message) {
  const searchResults = document.getElementById('search-results');
  if (searchResults) {
    searchResults.innerHTML = `<li class="search-status">${escapeHTML(message)}</li>`;
  }
}

function initSearch() {
  const searchRoot = document.getElementById('site-search');
  const searchToggle = document.getElementById('search-toggle');
  const searchInput = document.getElementById('search-input');
  const searchResults = document.getElementById('search-results');
  if (!searchRoot || !searchToggle || !searchInput || !searchResults) return;

  let searchTimeout = null;
  let isComposing = false;
  let requestID = 0;

  function setSearchOpen(open) {
    searchRoot.classList.toggle('site-search--open', open);
    searchToggle.setAttribute('aria-expanded', String(open));
    if (open) {
      window.requestAnimationFrame(() => searchInput.focus());
    }
  }

  async function performSearch() {
    const query = searchInput.value.trim();
    const currentRequest = ++requestID;

    if (query.length < 2) {
      searchResults.innerHTML = '';
      return;
    }

    showSearchStatus('Searching...');
    const results = await search(query);
    if (currentRequest === requestID) displayResults(results, query);
  }

  function scheduleSearch() {
    window.clearTimeout(searchTimeout);
    searchTimeout = window.setTimeout(performSearch, 160);
  }

  searchToggle.addEventListener('click', () => {
    setSearchOpen(!searchRoot.classList.contains('site-search--open'));
  });

  searchInput.addEventListener('input', () => {
    if (!isComposing) scheduleSearch();
  });

  searchInput.addEventListener('compositionstart', () => {
    isComposing = true;
  });

  searchInput.addEventListener('compositionend', () => {
    isComposing = false;
    scheduleSearch();
  });

  searchInput.addEventListener('keydown', (event) => {
    const items = Array.from(searchResults.querySelectorAll('.search-result-link'));
    if (event.key === 'ArrowDown' && items.length > 0) {
      event.preventDefault();
      items[0].focus();
    }
  });

  searchResults.addEventListener('keydown', (event) => {
    const items = Array.from(searchResults.querySelectorAll('.search-result-link'));
    const currentIndex = items.indexOf(document.activeElement);

    if (event.key === 'ArrowDown' && items.length > 0) {
      event.preventDefault();
      items[(currentIndex + 1) % items.length].focus();
    } else if (event.key === 'ArrowUp' && items.length > 0) {
      event.preventDefault();
      items[(currentIndex - 1 + items.length) % items.length].focus();
    }
  });

  document.addEventListener('pointerdown', (event) => {
    if (!searchRoot.contains(event.target)) setSearchOpen(false);
  });

  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape' && searchRoot.classList.contains('site-search--open')) {
      setSearchOpen(false);
      searchToggle.focus();
      return;
    }

    const target = event.target;
    const isTyping = target instanceof HTMLInputElement ||
      target instanceof HTMLTextAreaElement || target.isContentEditable;
    if (event.key === '/' && !isTyping && !event.metaKey && !event.ctrlKey && !event.altKey) {
      event.preventDefault();
      setSearchOpen(true);
    }
  });
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initSearch);
} else {
  initSearch();
}
