// Store node elements
const nodeElements = new Map();
const markers = {};
let map = null;
let ws = null;
let nodes = new Map();
let initialLoad = true; // Flag to track initial load

// Initialize WebSocket connection
function initializeWebSocket() {
    ws = new WebSocket(`ws://${window.location.host}/ws`);
    
    ws.onopen = () => {
        console.log('WebSocket connection established');
    };
    
    ws.onclose = () => {
        console.log('WebSocket connection closed');
        // Attempt to reconnect after 5 seconds
        setTimeout(initializeWebSocket, 5000);
    };
    
    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
    };
    
    ws.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data);
            if (Array.isArray(data)) {
                // Initial node list
                data.forEach(node => updateNode(node, true));
                initialLoad = false; // Mark initial load as complete
            } else {
                // Single node update
                updateNode(data, false);
            }
        } catch (error) {
            console.error('Error parsing WebSocket message:', error);
        }
    };
}

// Initialize the application
document.addEventListener('DOMContentLoaded', () => {
    // Initialize map
    const mapContainer = document.getElementById('map');
    if (mapContainer) {
        map = L.map('map').setView([0, 0], 2);
        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '© OpenStreetMap contributors'
        }).addTo(map);
    }

    // Initialize WebSocket connection
    initializeWebSocket();

    // Set dark theme by default
    document.documentElement.setAttribute('data-theme', 'dark');

    // Initialize mobile menu
    const mobileMenuBtn = document.querySelector('.mobile-menu-btn');
    const navLinks = document.querySelector('.nav-links');
    
    if (mobileMenuBtn && navLinks) {
        mobileMenuBtn.addEventListener('click', () => {
            navLinks.classList.toggle('show');
        });
    }

    // Initialize navigation
    const navItems = document.querySelectorAll('.nav-links a');
    navItems.forEach(item => {
        item.addEventListener('click', (e) => {
            e.preventDefault();
            
            // Update active states
            navItems.forEach(navItem => navItem.classList.remove('active'));
            item.classList.add('active');

            // Show corresponding section
            const targetSection = item.getAttribute('data-section');
            document.querySelectorAll('.content-section').forEach(section => {
                section.classList.remove('active');
                if (section.id === `${targetSection}-section`) {
                    section.classList.add('active');
                }
            });

            // Close mobile menu if open
            if (navLinks) {
                navLinks.classList.remove('show');
            }
        });
    });

    // Initialize search and filter
    initializeSearchAndFilter();
});

// Function to update a node
function updateNode(node, isInitialLoad) {
    console.log('Updating node:', node);
    
    // Check if this is a new node (not during initial load)
    const isNewNode = !isInitialLoad && !nodes.has(node.ip);
    
    // Store node data
    nodes.set(node.ip, node);
    
    // Update stats
    updateStats();
    
    // Update table if we're on the dashboard
    if (document.getElementById('nodeTableBody')) {
        updateTable();
    }
    
    // Update map if we're on the map page
    if (map) {
        updateMap();
    }

    // Show toast only for new nodes discovered after initial load
    if (isNewNode && !initialLoad) {
        const location = node.city && node.country ? 
            `${node.city}, ${node.country}` : 
            'Unknown location';
        
        showToast(`New node found: ${node.ip}<br>${location}`, 'success');
    }
}

// Seed node IPs
const SEED_NODES = [
    '95.217.43.225',
    '94.130.137.230',
    '95.217.42.247',
    '94.130.160.115',
    '195.201.107.230',
    '95.217.46.49',
    '159.69.76.144',
    '144.76.183.143'
];

function isSeedNode(ip) {
    return SEED_NODES.includes(ip);
}

// Function to update the table
function updateTable() {
    console.log('Updating table with', nodes.size, 'nodes');
    const tbody = document.getElementById('nodeTableBody');
    const searchTerm = document.getElementById('searchInput').value.toLowerCase();
    const statusFilter = document.getElementById('statusFilter').value;
    
    // Clear existing rows
    tbody.innerHTML = '';
    
    // Filter and display nodes
    nodes.forEach(node => {
        if (shouldDisplayNode(node, searchTerm, statusFilter)) {
            const row = createNodeTableRow(node);
            tbody.appendChild(row);
        }
    });
}

// Function to create a table row for a node
function createNodeTableRow(node) {
    const row = document.createElement('tr');
    row.className = 'node-row';
    row.dataset.ip = node.ip;
    
    // Status cell
    const statusCell = document.createElement('td');
    const statusIndicator = document.createElement('span');
    statusIndicator.className = `status-indicator ${node.isOnline ? 'status-online' : 'status-offline'}`;
    statusCell.appendChild(statusIndicator);
    row.appendChild(statusCell);

    // IP Address cell
    const ipCell = document.createElement('td');
    ipCell.className = 'status-ip-cell';
    const ipInfo = document.createElement('div');
    ipInfo.className = 'ip-info';
    const ipAddress = document.createElement('span');
    ipAddress.className = 'ip-address';
    ipAddress.textContent = node.ip;
    ipInfo.appendChild(ipAddress);
    ipCell.appendChild(ipInfo);
    row.appendChild(ipCell);

    // Location cell
    const locationCell = document.createElement('td');
    locationCell.innerHTML = `
        <div class="location-info">
            <div class="location-main">
                <i class="fas fa-globe"></i> ${node.country || 'Unknown'}
            </div>
            <div class="location-secondary">
                <i class="fas fa-city"></i> ${node.city || 'Unknown'}, ${node.regionName || 'Unknown'}
            </div>
        </div>
    `;
    row.appendChild(locationCell);

    // Network Info cell
    const networkCell = document.createElement('td');
    networkCell.innerHTML = `
        <div class="network-info">
            <span><i class="fas fa-network-wired"></i> ${node.isp || 'Unknown ISP'}</span>
            <span><i class="fas fa-server"></i> ${node.as || 'Unknown AS'}</span>
            <span><i class="fas fa-building"></i> ${node.org || 'Unknown Org'}</span>
        </div>
    `;
    row.appendChild(networkCell);

    // Tags cell
    const tagsCell = document.createElement('td');
    tagsCell.className = 'tags-cell';
    const tagsContainer = document.createElement('div');
    tagsContainer.className = 'tags-container';

    // Add seed node tag if applicable
    if (isSeedNode(node.ip)) {
        const seedTag = document.createElement('span');
        seedTag.className = 'tag seed-tag';
        seedTag.innerHTML = '<i class="fas fa-seedling"></i> Seed Node';
        tagsContainer.appendChild(seedTag);
    }

    // Add hosting tag if applicable
    if (node.hosting) {
        const hostingTag = document.createElement('span');
        hostingTag.className = 'tag hosting-tag';
        hostingTag.innerHTML = '<i class="fas fa-server"></i> Hosting';
        tagsContainer.appendChild(hostingTag);
    }

    // Add proxy tag if applicable
    if (node.proxy) {
        const proxyTag = document.createElement('span');
        proxyTag.className = 'tag proxy-tag';
        proxyTag.innerHTML = '<i class="fas fa-shield-alt"></i> Proxy';
        tagsContainer.appendChild(proxyTag);
    }

    // Add mobile tag if applicable
    if (node.mobile) {
        const mobileTag = document.createElement('span');
        mobileTag.className = 'tag mobile-tag';
        mobileTag.innerHTML = '<i class="fas fa-mobile-alt"></i> Mobile';
        tagsContainer.appendChild(mobileTag);
    }

    tagsCell.appendChild(tagsContainer);
    row.appendChild(tagsCell);

    // Last Seen cell
    const lastSeenCell = document.createElement('td');
    lastSeenCell.innerHTML = `
        <div class="last-seen">
            <i class="fas fa-clock"></i> ${formatDate(node.lastSeen)}
        </div>
    `;
    row.appendChild(lastSeenCell);

    // Actions cell
    const actionsCell = document.createElement('td');
    actionsCell.innerHTML = `
        <div class="action-buttons">
            <button class="action-button info-button" onclick="showNodeDetails('${node.ip}')" title="View Details">
                <i class="fas fa-info-circle"></i>
            </button>
        </div>
    `;
    row.appendChild(actionsCell);

    return row;
}

// Function to check if a node should be displayed
function shouldDisplayNode(node, searchTerm, statusFilter) {
    const matchesSearch = node.ip.toLowerCase().includes(searchTerm) ||
        (node.country && node.country.toLowerCase().includes(searchTerm)) ||
        (node.city && node.city.toLowerCase().includes(searchTerm));
    
    const matchesStatus = statusFilter === 'all' ||
        (statusFilter === 'online' && node.isOnline) ||
        (statusFilter === 'offline' && !node.isOnline);
    
    return matchesSearch && matchesStatus;
}

// Function to update stats
function updateStats() {
    const total = nodes.size;
    const online = Array.from(nodes.values()).filter(n => n.isOnline).length;
    const offline = total - online;
    const newNodes = Array.from(nodes.values()).filter(n => {
        const lastSeen = new Date(n.lastSeen);
        const now = new Date();
        return (now - lastSeen) < 24 * 60 * 60 * 1000; // Last 24 hours
    }).length;

    document.getElementById('totalNodes').textContent = total;
    document.getElementById('onlineNodes').textContent = online;
    document.getElementById('offlineNodes').textContent = offline;
    document.getElementById('newNodes').textContent = newNodes;
}

// Function to update the map
function updateMap() {
    if (!map) return;
    
    console.log('Updating map with', nodes.size, 'nodes');
    
    // Clear existing markers
    Object.values(markers).forEach(marker => marker.remove());
    markers = [];
    
    // Add new markers
    nodes.forEach(node => {
        if (node.lat && node.lon) {
            console.log(`Creating marker for node: ${node.ip} at coordinates:`, node.lat, node.lon);
            const marker = createMarker(node);
            if (marker) {
                markers[node.ip] = marker;
                marker.addTo(map);
            }
        } else {
            console.warn(`Missing coordinates for node ${node.ip}`);
        }
    });
}

// Function to create a marker
function createMarker(node) {
    if (!node.lat || !node.lon) return null;

    const lat = parseFloat(node.lat);
    const lon = parseFloat(node.lon);
    
    if (isNaN(lat) || isNaN(lon)) return null;

    const marker = L.marker([lat, lon], {
        icon: L.divIcon({
            className: `marker-pin ${node.isOnline ? 'online' : 'offline'}`,
            html: `<div class="marker-pin"></div>`,
            iconSize: [30, 30],
            iconAnchor: [15, 15]
        })
    });

    marker.bindPopup(createPopupContent(node));
    return marker;
}

// Function to create popup content
function createPopupContent(node) {
    return `
        <div class="popup-content">
            <h3>${node.ip}</h3>
            <p><strong>Status:</strong> ${node.isOnline ? 'Online' : 'Offline'}</p>
            <p><strong>Location:</strong> ${node.city || 'Unknown'}, ${node.country || 'Unknown'}</p>
            <p><strong>ISP:</strong> ${node.isp || 'Unknown'}</p>
            <p><strong>Last Seen:</strong> ${formatDate(node.lastSeen)}</p>
            <div class="popup-actions">
                <button onclick="showNodeDetails('${node.ip}')" class="popup-button">
                    <i class="fas fa-info-circle"></i> Details
                </button>
            </div>
        </div>
    `;
}

// Helper function to format dates
function formatDate(date) {
    if (!date) return 'Unknown';
    const d = new Date(date);
    return d.toLocaleString();
}

// Event listeners
document.addEventListener('DOMContentLoaded', () => {
    // Search input handler
    document.getElementById('searchInput').addEventListener('input', updateTable);
    
    // Status filter handler
    document.getElementById('statusFilter').addEventListener('change', updateTable);
    
    // Navigation handlers
    document.querySelectorAll('.nav-links a').forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            const section = e.target.dataset.section;
            
            // Update active link
            document.querySelectorAll('.nav-links a').forEach(l => l.classList.remove('active'));
            e.target.classList.add('active');
            
            // Show selected section
            document.querySelectorAll('.content-section').forEach(s => s.classList.remove('active'));
            document.getElementById(`${section}-section`).classList.add('active');
        });
    });
});

// Store markers and nodes
let selectedNode = null;

// Format time for display
function formatTime(timestamp) {
    if (!timestamp) return 'Never';
    const date = new Date(timestamp);
    return date.toLocaleString();
}

// Format uptime duration
function formatUptime(seconds) {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    
    if (days > 0) return `${days}d ${hours}h`;
    if (hours > 0) return `${hours}h ${minutes}m`;
    return `${minutes}m`;
}

// Notification system
function showNotification(title, message, type = 'info') {
    const container = document.getElementById('notificationContainer');
    if (!container) {
        console.error('Notification container not found');
        return;
    }
    const notification = document.createElement('div');
    notification.className = `notification ${type}`;
    
    notification.innerHTML = `
        <div class="notification-content">
            <div class="notification-title">${title}</div>
            <div class="notification-message">${message}</div>
        </div>
        <button class="notification-close">&times;</button>
    `;

    container.appendChild(notification);
    
    // Trigger animation
    setTimeout(() => {
        notification.classList.add('show');
    }, 10);

    // Add close button functionality
    const closeButton = notification.querySelector('.notification-close');
    closeButton.addEventListener('click', () => {
        notification.classList.remove('show');
        setTimeout(() => {
            notification.remove();
        }, 300);
    });

    // Auto-remove after 5 seconds
    setTimeout(() => {
        if (notification.parentElement) {
            notification.classList.remove('show');
            setTimeout(() => {
                notification.remove();
            }, 300);
        }
    }, 5000);
}

function showNodeDetails(ip) {
    const node = nodes.get(ip);
    if (!node) return;

    const modal = document.createElement('div');
    modal.className = 'modal';
    modal.innerHTML = `
        <div class="modal-content">
            <div class="modal-header">
                <h3>Node Details</h3>
                <button class="close-button">&times;</button>
            </div>
            <div class="modal-body">
                <div class="details-section">
                    <h4>Status & Network</h4>
                    <div class="details-grid">
                        <div class="detail-item">
                            <span class="detail-label">Status</span>
                            <span class="detail-value">${node.isOnline ? 'Online' : 'Offline'}</span>
                        </div>
                        <div class="detail-item">
                            <span class="detail-label">ISP</span>
                            <span class="detail-value">${node.isp || 'Unknown'}</span>
                        </div>
                        <div class="detail-item">
                            <span class="detail-label">AS</span>
                            <span class="detail-value">${node.as || 'Unknown'}</span>
                        </div>
                        <div class="detail-item">
                            <span class="detail-label">Last Seen</span>
                            <span class="detail-value">${formatDate(node.lastSeen)}</span>
                        </div>
                    </div>
                </div>
                <div class="details-section">
                    <h4>Location</h4>
                    <div class="details-grid">
                        <div class="detail-item">
                            <span class="detail-label">Country</span>
                            <span class="detail-value">${node.country || 'Unknown'}</span>
                        </div>
                        <div class="detail-item">
                            <span class="detail-label">City</span>
                            <span class="detail-value">${node.city || 'Unknown'}</span>
                        </div>
                        <div class="detail-item">
                            <span class="detail-label">Region</span>
                            <span class="detail-value">${node.regionName || 'Unknown'}</span>
                        </div>
                        <div class="detail-item">
                            <span class="detail-label">Timezone</span>
                            <span class="detail-value">${node.timezone || 'Unknown'}</span>
                        </div>
                    </div>
                </div>
                <div class="details-section">
                    <h4>Node Tags</h4>
                    <div class="tags">
                        ${node.mobile ? '<span class="tag mobile-tag">Mobile</span>' : ''}
                        ${node.proxy ? '<span class="tag proxy-tag">Proxy</span>' : ''}
                        ${node.hosting ? '<span class="tag hosting-tag">Hosting</span>' : ''}
                    </div>
                </div>
            </div>
        </div>
    `;

    document.body.appendChild(modal);
    setTimeout(() => modal.classList.add('show'), 10);

    // Close modal handlers
    const closeButton = modal.querySelector('.close-button');
    closeButton.onclick = () => closeModal(modal);

    modal.onclick = (e) => {
        if (e.target === modal) closeModal(modal);
    };

    document.addEventListener('keydown', function closeOnEscape(e) {
        if (e.key === 'Escape') {
            closeModal(modal);
            document.removeEventListener('keydown', closeOnEscape);
        }
    });
}

function closeModal(modal) {
    modal.classList.remove('show');
    setTimeout(() => modal.remove(), 300);
}

function showOnMap(ip) {
    const node = nodes.get(ip);
    if (!node || !node.lat || !node.lon) return;

    const lat = parseFloat(node.lat);
    const lon = parseFloat(node.lon);
    
    if (isNaN(lat) || isNaN(lon)) return;

    // Center the map on the node
    map.setView([lat, lon], 8);

    // Find and highlight the marker
    if (markers[ip]) {
        markers[ip].openPopup();
        const icon = markers[ip].getIcon();
        icon.options.className += ' highlight';
        markers[ip].setIcon(icon);
        setTimeout(() => {
            icon.options.className = icon.options.className.replace(' highlight', '');
            markers[ip].setIcon(icon);
        }, 2000);
    }
}

// Initialize map if we're on the map page
function initializeMap() {
    if (document.getElementById('map')) {
        map = L.map('map').setView([0, 0], 2);
        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '© OpenStreetMap contributors'
        }).addTo(map);
    }
}

// Add event listeners for map filters
const filters = ['showOnline', 'showOffline', 'showNew'];
filters.forEach(filterId => {
    const checkbox = document.getElementById(filterId);
    if (checkbox) {
        checkbox.addEventListener('change', updateMap);
    }
});

// Function to show toast notification
function showToast(message, type = 'info') {
    // Create toast container if it doesn't exist
    let container = document.querySelector('.toast-container');
    if (!container) {
        container = document.createElement('div');
        container.className = 'toast-container';
        document.body.appendChild(container);
    }

    // Create toast element
    const toast = document.createElement('div');
    toast.className = 'toast';
    
    // Set icon based on type
    let icon = 'fa-info-circle';
    if (type === 'success') icon = 'fa-check-circle';
    if (type === 'error') icon = 'fa-exclamation-circle';
    if (type === 'warning') icon = 'fa-exclamation-triangle';

    toast.innerHTML = `
        <div class="toast-content">
            <i class="fas ${icon} toast-icon"></i>
            <div class="toast-message">${message}</div>
        </div>
        <button class="toast-close">&times;</button>
    `;

    // Add to container
    container.appendChild(toast);

    // Show toast with animation
    setTimeout(() => toast.classList.add('show'), 10);

    // Add close button functionality
    const closeButton = toast.querySelector('.toast-close');
    closeButton.addEventListener('click', () => {
        toast.classList.remove('show');
        setTimeout(() => toast.remove(), 300);
    });

    // Auto-remove after 5 seconds
    setTimeout(() => {
        if (toast.parentElement) {
            toast.classList.remove('show');
            setTimeout(() => toast.remove(), 300);
        }
    }, 5000);
}

// Function to export nodes to a text file
function exportNodes() {
    // Get the current filter value
    const statusFilter = document.getElementById('statusFilter').value;
    
    // Filter nodes based on current filter
    const filteredNodes = Array.from(nodes.values()).filter(node => {
        if (statusFilter === 'all') return true;
        if (statusFilter === 'online') return node.isOnline;
        if (statusFilter === 'offline') return !node.isOnline;
        return true;
    });

    // Create the content
    let content = '# Zano Network Nodes\n';
    content += `# Exported on: ${new Date().toLocaleString()}\n`;
    content += `# Total Nodes: ${filteredNodes.length}\n`;
    content += `# Online Nodes: ${filteredNodes.filter(n => n.isOnline).length}\n`;
    content += `# Offline Nodes: ${filteredNodes.filter(n => !n.isOnline).length}\n\n`;

    // Add node information
    filteredNodes.forEach(node => {
        content += `# ${node.ip}\n`;
        content += `# Status: ${node.isOnline ? 'Online' : 'Offline'}\n`;
        content += `# Location: ${node.city || 'Unknown'}, ${node.country || 'Unknown'}\n`;
        content += `# ISP: ${node.isp || 'Unknown'}\n`;
        content += `# Last Seen: ${formatDate(node.lastSeen)}\n`;
        
        // Add tags
        const tags = [];
        if (isSeedNode(node.ip)) tags.push('Seed Node');
        if (node.hosting) tags.push('Hosting');
        if (node.proxy) tags.push('Proxy');
        if (node.mobile) tags.push('Mobile');
        if (tags.length > 0) {
            content += `# Tags: ${tags.join(', ')}\n`;
        }
        
        // Add the IP address for easy copying
        content += `${node.ip}\n\n`;
    });

    // Create and download the file
    const blob = new Blob([content], { type: 'text/plain' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `zano-nodes-${new Date().toISOString().split('T')[0]}.txt`;
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(url);
    document.body.removeChild(a);

    // Show success notification
    showToast(`Exported ${filteredNodes.length} nodes to file`, 'success');
}

// Add event listener for export button
document.addEventListener('DOMContentLoaded', () => {
    const exportButton = document.getElementById('exportNodes');
    if (exportButton) {
        exportButton.addEventListener('click', exportNodes);
    }
}); 