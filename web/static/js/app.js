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
            destroyTerminal();
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

    // Start SSE after Alpine store is ready
    initSSE();

    // Wire up search input debounce
    const searchInput = document.getElementById('search-input');
    if (searchInput) {
        searchInput.addEventListener('input', debounce(function () {
            Alpine.store('app').searchQuery = this.value;
            Alpine.store('app').refreshContainers();
        }, 300));
    }
});

// SSE connection management with auto-reconnect
function initSSE() {
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
            Alpine.store('app').sseConnected = true;
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
            Alpine.store('app').sseConnected = false;
            eventSource.close();
            // Exponential backoff reconnect
            const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), maxReconnectDelay);
            reconnectAttempts++;
            setTimeout(connect, delay);
        };
    }

    connect();
}

// Search debounce helper
function debounce(fn, delay) {
    let timer;
    return function (...args) {
        clearTimeout(timer);
        timer = setTimeout(() => fn.apply(this, args), delay);
    };
}

// Terminal management
let activeTerminal = null;
let activeTerminalWS = null;

function initTerminal(containerId) {
    // Clean up any existing terminal
    destroyTerminal();

    const container = document.getElementById('terminal-container');
    if (!container) return;

    // Check if xterm is available
    if (typeof Terminal === 'undefined') return;

    const term = new Terminal({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: 'Menlo, Monaco, "Courier New", monospace',
        theme: {
            background: document.documentElement.classList.contains('dark') ? '#1a1a2e' : '#1e1e1e',
            foreground: '#e0e0e0',
        },
    });

    const fitAddon = new FitAddon.FitAddon();
    term.loadAddon(fitAddon);
    term.open(container);
    fitAddon.fit();

    // WebSocket connection
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(proto + '//' + location.host + '/ws/terminal?id=' + containerId);

    ws.onopen = function () {
        // Send initial resize
        const dims = { type: 'resize', cols: term.cols, rows: term.rows };
        ws.send(new Blob([JSON.stringify(dims)]));
    };

    ws.onmessage = function (e) {
        if (typeof e.data === 'string') {
            term.write(e.data);
        } else if (e.data instanceof Blob) {
            e.data.text().then(text => term.write(text));
        }
    };

    ws.onclose = function () {
        term.write('\r\n\x1b[31m[Session ended]\x1b[0m\r\n');
    };

    ws.onerror = function () {
        term.write('\r\n\x1b[31m[Connection error]\x1b[0m\r\n');
    };

    // stdin: terminal → WebSocket
    term.onData(function (data) {
        if (ws.readyState === WebSocket.OPEN) {
            ws.send(data);
        }
    });

    // Handle resize
    term.onResize(function (size) {
        if (ws.readyState === WebSocket.OPEN) {
            const dims = { type: 'resize', cols: size.cols, rows: size.rows };
            ws.send(new Blob([JSON.stringify(dims)]));
        }
    });

    // Resize on window resize
    const resizeHandler = function () {
        fitAddon.fit();
    };
    window.addEventListener('resize', resizeHandler);

    activeTerminal = { term, fitAddon, resizeHandler };
    activeTerminalWS = ws;
}

function destroyTerminal() {
    if (activeTerminalWS) {
        activeTerminalWS.close();
        activeTerminalWS = null;
    }
    if (activeTerminal) {
        window.removeEventListener('resize', activeTerminal.resizeHandler);
        activeTerminal.term.dispose();
        activeTerminal = null;
    }
}
