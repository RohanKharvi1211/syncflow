// API Base URL
const API_BASE_URL = 'http://localhost:8080/api';

// State
let currentCompany = null;
let apps = [];
let connections = [];

// Initialize on page load
document.addEventListener('DOMContentLoaded', async () => {
    await loadApps();
    await loadConnections();
    renderConnectors();
    setupEventListeners();
});

// Load apps from API
async function loadApps() {
    try {
        const response = await fetch(`${API_BASE_URL}/apps`);
        const data = await response.json();
        apps = data.apps || [];
    } catch (error) {
        console.error('Error loading apps:', error);
        // Fallback to default apps if API fails
        apps = getDefaultApps();
    }
}

// Default apps if API is not available
function getDefaultApps() {
    return [
        { id: '1', name: 'salesforce', display_name: 'Salesforce', description: 'CRM / TMS', type: 'both' },
        { id: '2', name: 'quickbooks', display_name: 'QuickBooks Online', description: 'Accounting', type: 'destination' },
        { id: '3', name: 'shopify', display_name: 'Shopify', description: 'E-Commerce', type: 'both' },
        { id: '4', name: 'hubspot', display_name: 'HubSpot', description: 'Marketing', type: 'both' },
        { id: '5', name: 'xero', display_name: 'Xero', description: 'Accounting', type: 'destination' },
    ];
}

// Load connections from API
async function loadConnections() {
    try {
        // For now, we'll use a mock company ID
        // In production, this would come from the logged-in user's session
        const companyId = localStorage.getItem('companyId');
        if (companyId) {
            const response = await fetch(`${API_BASE_URL}/connections?company_id=${companyId}`);
            const data = await response.json();
            connections = data.connections || [];
        }
    } catch (error) {
        console.error('Error loading connections:', error);
        connections = [];
    }
}

// Render connector cards
function renderConnectors() {
    const grid = document.getElementById('connectors-grid');
    grid.innerHTML = '';

    apps.forEach(app => {
        const card = createConnectorCard(app);
        grid.appendChild(card);
    });
}

// Create connector card element
function createConnectorCard(app) {
    const card = document.createElement('div');
    card.className = 'connector-card';
    card.onclick = () => handleConnectorClick(app);

    const iconClass = getIconClass(app.name);
    const iconLetter = getIconLetter(app.name);

    card.innerHTML = `
        <div class="connector-icon ${iconClass}">${iconLetter}</div>
        <div class="connector-name">${app.display_name || app.name}</div>
        <div class="connector-type">${app.description || app.type}</div>
    `;

    return card;
}

// Get icon class for app
function getIconClass(appName) {
    const name = appName.toLowerCase();
    if (name.includes('salesforce')) return 'salesforce';
    if (name.includes('quickbook')) return 'quickbooks';
    if (name.includes('shopify')) return 'shopify';
    if (name.includes('hubspot')) return 'hubspot';
    if (name.includes('xero')) return 'xero';
    if (name.includes('google')) return 'googlesheet';
    if (name.includes('tally')) return 'tally';
    return 'salesforce'; // default
}

// Get icon letter for app
function getIconLetter(appName) {
    const name = appName.toLowerCase();
    if (name.includes('salesforce')) return 'S';
    if (name.includes('quickbook')) return 'Q';
    if (name.includes('shopify')) return 'S';
    if (name.includes('hubspot')) return 'H';
    if (name.includes('xero')) return 'X';
    if (name.includes('google')) return 'G';
    if (name.includes('tally')) return 'T';
    return appName.charAt(0).toUpperCase();
}

// Handle connector card click
function handleConnectorClick(app) {
    // For now, just open the create connection modal
    // In the future, this could show app details or initiate connection
    openCreateConnectionModal();
}

// Toggle connection details
function toggleConnection(connectionId) {
    const details = document.getElementById(connectionId);
    const arrow = document.getElementById(`arrow-${connectionId}`);
    
    if (details.classList.contains('hidden')) {
        details.classList.remove('hidden');
        arrow.classList.add('rotated');
    } else {
        details.classList.add('hidden');
        arrow.classList.remove('rotated');
    }
}

// Open create connection modal
function openCreateConnectionModal() {
    const modal = document.getElementById('create-connection-modal');
    modal.classList.remove('hidden');
    
    // Populate app selects
    populateAppSelects();
}

// Close create connection modal
function closeCreateConnectionModal() {
    const modal = document.getElementById('create-connection-modal');
    modal.classList.add('hidden');
    document.getElementById('create-connection-form').reset();
}

// Populate app selects
function populateAppSelects() {
    const sourceSelect = document.getElementById('source-app');
    const destSelect = document.getElementById('destination-app');
    
    // Clear existing options (except first)
    sourceSelect.innerHTML = '<option value="">Select source app...</option>';
    destSelect.innerHTML = '<option value="">Select destination app...</option>';
    
    // Filter apps by type
    const sourceApps = apps.filter(app => app.type === 'source' || app.type === 'both');
    const destApps = apps.filter(app => app.type === 'destination' || app.type === 'both');
    
    sourceApps.forEach(app => {
        const option = document.createElement('option');
        option.value = app.id;
        option.textContent = app.display_name || app.name;
        sourceSelect.appendChild(option);
    });
    
    destApps.forEach(app => {
        const option = document.createElement('option');
        option.value = app.id;
        option.textContent = app.display_name || app.name;
        destSelect.appendChild(option);
    });
}

// Setup event listeners
function setupEventListeners() {
    // Navigation items
    document.querySelectorAll('.nav-item').forEach(item => {
        item.addEventListener('click', (e) => {
            e.preventDefault();
            const page = item.dataset.page;
            handleNavigation(page);
        });
    });
    
    // Create connection form
    document.getElementById('create-connection-form').addEventListener('submit', handleCreateConnection);
}

// Handle navigation
function handleNavigation(page) {
    // Remove active class from all nav items
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.remove('active');
    });
    
    // Add active class to clicked item
    event.target.closest('.nav-item').classList.add('active');
    
    // Handle different pages
    switch(page) {
        case 'marketplace':
            // Already on marketplace
            break;
        case 'exceptions':
            alert('Exception Center - Coming soon!');
            break;
        case 'notifications':
            alert('Notifications - Coming soon!');
            break;
    }
}

// Handle create connection form submission
async function handleCreateConnection(e) {
    e.preventDefault();
    
    const formData = new FormData(e.target);
    const sourceAppId = formData.get('source_app');
    const destAppId = formData.get('destination_app');
    const name = formData.get('name');
    
    // Get company ID from localStorage or use a default
    const companyId = localStorage.getItem('companyId') || 'default-company-id';
    
    try {
        const response = await fetch(`${API_BASE_URL}/integrations`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                company_id: companyId,
                name: name,
                source_app_id: sourceAppId,
                destination_app_id: destAppId,
                status: 'active'
            })
        });
        
        if (response.ok) {
            const data = await response.json();
            alert('Connection created successfully!');
            closeCreateConnectionModal();
            await loadConnections();
        } else {
            const error = await response.json();
            alert(`Error: ${error.error || 'Failed to create connection'}`);
        }
    } catch (error) {
        console.error('Error creating connection:', error);
        alert('Failed to create connection. Please try again.');
    }
}

// Update tenant name
function updateTenantName(name) {
    document.getElementById('tenant-name').textContent = name;
    localStorage.setItem('tenantName', name);
}

// Load tenant name from localStorage on page load
const savedTenantName = localStorage.getItem('tenantName');
if (savedTenantName) {
    updateTenantName(savedTenantName);
}




