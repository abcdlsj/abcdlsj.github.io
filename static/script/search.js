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
    
    return Array.from(results);
}

document.addEventListener('DOMContentLoaded', () => {
    const searchInput = document.getElementById('search-input');
    const searchResults = document.getElementById('search-results');
    
    searchInput.addEventListener('input', async () => {
        const query = searchInput.value.trim();
        if (query.length < 3) {
            searchResults.innerHTML = '';
            return;
        }
        
        const results = await search(query);
        searchResults.innerHTML = results.map(url => `<li><a href="/posts/${url}.html">${url}</a></li>`).join('');
    });
});