// WingStation Alpine.js stores and SSE management

document.addEventListener('alpine:init', () => {
    // Global application state
    Alpine.store('app', {
        sseConnected: false,
        searchQuery: '',
        statusFilter: '',
        groupFilter: '',
        sortBy: 'priority',
        detailOpen: false,
        detailContainerId: null,

        openDetail(containerId) {
            this.detailContainerId = containerId;
            this.detailOpen = true;
            htmx.ajax('GET', '/partials/detail?id=' + containerId, '#detail-content');
        },

        closeDetail() {
            this.detailOpen = false;
            this.detailContainerId = null;
        },

        setFilter(status) {
            this.statusFilter = this.statusFilter === status ? '' : status;
            this.refreshContainers();
        },

        setGroup(group) {
            this.groupFilter = group;
            this.refreshContainers();
        },

        setSort(sort) {
            this.sortBy = sort;
            this.refreshContainers();
        },

        refreshContainers() {
            const params = new URLSearchParams();
            if (this.searchQuery) params.set('q', this.searchQuery);
            if (this.statusFilter) params.set('status', this.statusFilter);
            if (this.groupFilter) params.set('group', this.groupFilter);
            if (this.sortBy) params.set('sort', this.sortBy);
            htmx.ajax('GET', '/partials/containers?' + params.toString(), '#container-list');
        }
    });
});

// SSE connection management with auto-reconnect
(function () {
    let eventSource = null;
    let reconnectAttempts = 0;
    const maxReconnectDelay = 30000;

    function connect() {
        if (eventSource) {
            eventSource.close();
        }

        eventSource = new EventSource('/events');

        eventSource.onopen = function () {
            reconnectAttempts = 0;
            if (window.Alpine) {
                Alpine.store('app').sseConnected = true;
            }
        };

        eventSource.addEventListener('containers', function (e) {
            const target = document.getElementById('container-list');
            if (target) {
                target.innerHTML = e.data;
                htmx.process(target);
            }
        });

        eventSource.addEventListener('container-update', function (e) {
            const data = JSON.parse(e.data);
            const card = document.getElementById('container-' + data.id);
            if (card) {
                htmx.ajax('GET', '/partials/container-card?id=' + data.id, card);
            }
        });

        eventSource.onerror = function () {
            if (window.Alpine) {
                Alpine.store('app').sseConnected = false;
            }
            eventSource.close();
            // Exponential backoff reconnect
            const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), maxReconnectDelay);
            reconnectAttempts++;
            setTimeout(connect, delay);
        };
    }

    // Start SSE connection when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', connect);
    } else {
        connect();
    }
})();

// Search debounce helper
function debounce(fn, delay) {
    let timer;
    return function (...args) {
        clearTimeout(timer);
        timer = setTimeout(() => fn.apply(this, args), delay);
    };
}

// Wire up search input debounce
document.addEventListener('DOMContentLoaded', function () {
    const searchInput = document.getElementById('search-input');
    if (searchInput) {
        searchInput.addEventListener('input', debounce(function () {
            if (window.Alpine) {
                Alpine.store('app').searchQuery = this.value;
                Alpine.store('app').refreshContainers();
            }
        }, 300));
    }
});
