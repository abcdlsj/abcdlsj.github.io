let searchIndex = null;

async function loadSearchIndex() {
    if (searchIndex === null) {
        const response = await fetch('/search-index.json');
        searchIndex = await response.json();
    }
}

async function search(query) {
    await loadSearchIndex();
    const words = query.toLowerCase().split(/\s+/);
    const results = new Set();
    
    for (const word of words) {
        if (word in searchIndex.words) {
            for (const postUrl of searchIndex.words[word]) {
                results.add(postUrl);
            }
        }
    }

    console.log(results);
    console.log(query);
    
    return Array.from(results);
}

let searchTimeout = null;
let isComposing = false;

document.addEventListener('DOMContentLoaded', () => {
    const searchInput = document.getElementById('search-input');
    const searchResults = document.getElementById('search-results');
    
    searchInput.addEventListener('input', () => {
        if (isComposing) return;
        scheduleSearch();
    });

    searchInput.addEventListener('compositionstart', () => {
        isComposing = true;
    });

    searchInput.addEventListener('compositionend', () => {
        isComposing = false;
        scheduleSearch();
    });

    function scheduleSearch() {
        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => performSearch(), 300);
    }

    async function performSearch() {
        const query = searchInput.value.trim();
        if (query.length < 2) {
            searchResults.innerHTML = '';
            return;
        }

        const results = await search(query);
        searchResults.innerHTML = results.map(url => `<li><a href="/posts/${url}.html">${url}</a></li>`).join('');
    }
});