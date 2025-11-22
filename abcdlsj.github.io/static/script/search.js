/**
 * 搜索功能 - Enhanced Search Functionality
 * 提供流畅的搜索体验和更好的用户界面
 */

let searchIndex = null;
let searchCache = new Map();

/**
 * 加载搜索索引
 */
async function loadSearchIndex() {
    if (searchIndex === null) {
        try {
            const response = await fetch('/search-index.json');
            searchIndex = await response.json();
            console.log('搜索索引加载完成');
        } catch (error) {
            console.error('加载搜索索引失败:', error);
            searchIndex = { words: {}, posts: {} };
        }
    }
}

/**
 * 执行搜索
 */
async function search(query) {
    await loadSearchIndex();
    
    // 检查缓存
    if (searchCache.has(query)) {
        return searchCache.get(query);
    }
    
    const words = query.toLowerCase().split(/\s+/).filter(word => word.length > 0);
    const results = new Map();
    
    for (const word of words) {
        if (word in searchIndex.words) {
            for (const postUrl of searchIndex.words[word]) {
                const currentCount = results.get(postUrl) || 0;
                results.set(postUrl, currentCount + 1);
            }
        }
    }
    
    // 按匹配度排序
    const sortedResults = Array.from(results.entries())
        .sort((a, b) => b[1] - a[1])
        .map(([url]) => url);
    
    // 缓存结果
    searchCache.set(query, sortedResults);
    
    return sortedResults;
}

/**
 * 获取文章标题
 */
function getPostTitle(url) {
    if (searchIndex && searchIndex.posts && searchIndex.posts[url]) {
        return searchIndex.posts[url].title || url;
    }
    return url;
}

/**
 * 高亮搜索关键词
 */
function highlightText(text, query) {
    if (!text || !query) return text;
    
    const words = query.toLowerCase().split(/\s+/).filter(word => word.length > 0);
    let highlightedText = text;
    
    words.forEach(word => {
        const regex = new RegExp(`(${word})`, 'gi');
        highlightedText = highlightedText.replace(regex, '<mark>$1</mark>');
    });
    
    return highlightedText;
}

/**
 * 显示搜索结果
 */
function displayResults(results, query) {
    const searchResults = document.getElementById('search-results');
    
    if (results.length === 0) {
        searchResults.innerHTML = '<li class="search-no-results">未找到相关文章</li>';
        return;
    }
    
    const resultsHTML = results.slice(0, 8).map(url => {
        const title = getPostTitle(url);
        const highlightedTitle = highlightText(title, query);
        
        return `
            <li class="search-result-item">
                <a href="/posts/${url}.html" class="search-result-link">
                    ${highlightedTitle}
                </a>
            </li>
        `;
    }).join('');
    
    searchResults.innerHTML = resultsHTML;
}

/**
 * 显示搜索状态
 */
function showSearchStatus(message) {
    const searchResults = document.getElementById('search-results');
    searchResults.innerHTML = `<li class="search-status">${message}</li>`;
}

// 搜索控制变量
let searchTimeout = null;
let isComposing = false;
let isSearching = false;

/**
 * 初始化搜索功能
 */
document.addEventListener('DOMContentLoaded', () => {
    const searchInput = document.getElementById('search-input');
    const searchResults = document.getElementById('search-results');
    
    if (!searchInput || !searchResults) {
        console.log('搜索元素未找到，跳过搜索初始化');
        return;
    }
    
    // 输入事件处理
    searchInput.addEventListener('input', () => {
        if (isComposing || isSearching) return;
        scheduleSearch();
    });

    // 输入法组合事件
    searchInput.addEventListener('compositionstart', () => {
        isComposing = true;
    });

    searchInput.addEventListener('compositionend', () => {
        isComposing = false;
        scheduleSearch();
    });

    // 键盘导航
    searchInput.addEventListener('keydown', (e) => {
        const items = searchResults.querySelectorAll('.search-result-link');
        const currentIndex = Array.from(items).findIndex(item => item === document.activeElement);
        
        if (e.key === 'ArrowDown' && items.length > 0) {
            e.preventDefault();
            const nextIndex = currentIndex < items.length - 1 ? currentIndex + 1 : 0;
            items[nextIndex].focus();
        } else if (e.key === 'ArrowUp' && items.length > 0) {
            e.preventDefault();
            const prevIndex = currentIndex > 0 ? currentIndex - 1 : items.length - 1;
            items[prevIndex].focus();
        } else if (e.key === 'Escape') {
            searchInput.value = '';
            searchResults.innerHTML = '';
            searchInput.blur();
        }
    });

    // 失去焦点时隐藏结果（延迟以允许点击）
    searchInput.addEventListener('blur', () => {
        setTimeout(() => {
            if (!searchResults.contains(document.activeElement)) {
                searchResults.innerHTML = '';
            }
        }, 200);
    });

    /**
     * 调度搜索
     */
    function scheduleSearch() {
        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => performSearch(), 200);
    }

    /**
     * 执行搜索
     */
    async function performSearch() {
        const query = searchInput.value.trim();
        
        if (query.length < 2) {
            searchResults.innerHTML = '';
            return;
        }
        
        if (isSearching) return;
        isSearching = true;
        
        showSearchStatus('搜索中...');
        
        try {
            const results = await search(query);
            displayResults(results, query);
        } catch (error) {
            console.error('搜索失败:', error);
            showSearchStatus('搜索失败，请重试');
        } finally {
            isSearching = false;
        }
    }
});