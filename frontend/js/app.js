const API_BASE_URL = 'http://localhost:8080/api';

let users = [];
let currentRole = null;
let currentUser = null;
let selectedBackofficeUserId = null;
let selectedIntegrationId = null;

document.addEventListener('DOMContentLoaded', () => {
    initializeApp();
});

async function initializeApp() {
    showLoading();
    await loadUsers();
    populateLoginSelect();
    setupEventListeners();
    showLogin();
    hideLoading();
}

function setupEventListeners() {
    document.getElementById('login-form').addEventListener('submit', handleLogin);

    document.querySelectorAll('input[name="role"]').forEach(radio => {
        radio.addEventListener('change', () => {
            const role = document.querySelector('input[name="role"]:checked').value;
            document.getElementById('user-select-group').style.display = role === 'user' ? 'block' : 'none';
        });
    });
}

async function loadUsers() {
    try {
        const response = await fetch(`${API_BASE_URL}/users`);
        const data = await response.json();
        users = data.users || [];
    } catch (error) {
        console.error('Failed to load users', error);
        showNotification('Failed to load companies', 'error');
    }
}

function populateLoginSelect() {
    const select = document.getElementById('user-select');
    if (!users.length) {
        select.innerHTML = '<option value="">No companies configured</option>';
        return;
    }

    select.innerHTML = '<option value="">Select your company...</option>';
    users.forEach(user => {
        const option = document.createElement('option');
        option.value = user.id;
        option.textContent = user.company_name || user.email;
        select.appendChild(option);
    });
}

function showLogin() {
    document.getElementById('login-screen').classList.remove('hidden');
    document.getElementById('dashboard').classList.add('hidden');
    document.getElementById('logout-btn').classList.add('hidden');
}

function showDashboard() {
    document.getElementById('login-screen').classList.add('hidden');
    document.getElementById('dashboard').classList.remove('hidden');
    document.getElementById('logout-btn').classList.remove('hidden');
    document.getElementById('last-refresh-label').textContent = new Date().toLocaleString();
}

function handleLogin(event) {
    event.preventDefault();
    const role = document.querySelector('input[name="role"]:checked').value;

    if (role === 'user') {
        const userId = document.getElementById('user-select').value;
        if (!userId) {
            showNotification('Please select your company', 'error');
            return;
        }
        const user = users.find(u => u.id === userId);
        if (!user) {
            showNotification('Selected company not found', 'error');
            return;
        }
        currentRole = 'user';
        currentUser = user;
        document.getElementById('current-role-label').textContent = 'Client / User';
        document.getElementById('current-user-label').textContent = user.company_name || user.email;
        enterUserView();
    } else {
        currentRole = 'backoffice';
        currentUser = null;
        document.getElementById('current-role-label').textContent = 'Backoffice';
        document.getElementById('current-user-label').textContent = 'All companies';
        enterBackofficeView();
    }

    showDashboard();
}

function logout() {
    currentRole = null;
    currentUser = null;
    selectedBackofficeUserId = null;
    selectedIntegrationId = null;
    showLogin();
    document.getElementById('login-form').reset();
    document.getElementById('user-select-group').style.display = 'block';
    showNotification('Logged out', 'info');
}

function enterBackofficeView() {
    selectedBackofficeUserId = null;
    selectedIntegrationId = null;
    document.getElementById('backoffice-view').classList.remove('hidden');
    document.getElementById('user-view').classList.add('hidden');
    renderCompanyList();
    document.getElementById('integration-hint').textContent = 'Select a company to view integrations';
    document.getElementById('record-hint').textContent = 'Select an integration to view sync records';
    document.getElementById('backoffice-integrations').innerHTML = '';
    document.getElementById('backoffice-records').innerHTML = '';
}

function renderCompanyList() {
    const container = document.getElementById('company-list');
    if (!users.length) {
        container.innerHTML = '<p class="muted">No companies configured.</p>';
        return;
    }

    container.innerHTML = '';
    users.forEach(user => {
        const button = document.createElement('button');
        button.textContent = user.company_name || user.email;
        if (user.id === selectedBackofficeUserId) {
            button.classList.add('active');
        }
        button.addEventListener('click', () => selectBackofficeCompany(user.id));
        container.appendChild(button);
    });
}

async function selectBackofficeCompany(userId) {
    selectedBackofficeUserId = userId;
    renderCompanyList();
    await loadIntegrationsForUser(userId, 'backoffice');
    document.getElementById('record-hint').textContent = 'Select an integration to view sync records';
}

async function enterUserView() {
    selectedIntegrationId = null;
    document.getElementById('backoffice-view').classList.add('hidden');
    document.getElementById('user-view').classList.remove('hidden');
    await loadIntegrationsForUser(currentUser.id, 'user');
}

async function loadIntegrationsForUser(userId, scope) {
    try {
        showLoading();
        const response = await fetch(`${API_BASE_URL}/integrations?user_id=${userId}`);
        const data = await response.json();
        const integrations = data.integrations || [];
        renderIntegrations(integrations, scope);
    } catch (error) {
        console.error('Failed to load integrations', error);
        showNotification('Failed to load integrations', 'error');
    } finally {
        hideLoading();
    }
}

function renderIntegrations(integrations, scope) {
    const containerId = scope === 'backoffice' ? 'backoffice-integrations' : 'user-integrations';
    const container = document.getElementById(containerId);

    if (!integrations.length) {
        container.innerHTML = '<p class="muted">No integrations yet.</p>';
        if (scope === 'backoffice') {
            document.getElementById('integration-hint').textContent = 'No integrations configured for this company.';
        }
        if (scope === 'user') {
            document.getElementById('user-records').innerHTML = '<p class="muted">Select an integration to see its records.</p>';
        }
        return;
    }

    container.innerHTML = '';
    integrations.forEach(integration => {
        const card = document.createElement('div');
        card.className = 'integration-card';
        const status = integration.status || 'active';
        card.innerHTML = `
            <div class="card-header">
                <h3>${integration.name}</h3>
                <span class="status-badge ${statusClass(status)}">${status}</span>
            </div>
            <p><strong>Source:</strong> Google Sheets</p>
            <p><strong>Destination:</strong> QuickBooks (${integration.destination_entity || 'N/A'})</p>
            <p><strong>Sync every:</strong> ${integration.sync_frequency_minutes || 60} min</p>
            <button class="btn btn-secondary" onclick="viewRecords('${integration.id}', '${scope}')">
                <i class="fas fa-list"></i> View Records
            </button>
        `;
        container.appendChild(card);
    });
}

async function viewRecords(integrationId, scope) {
    selectedIntegrationId = integrationId;
    const targetId = scope === 'backoffice' ? 'backoffice-records' : 'user-records';
    const container = document.getElementById(targetId);
    container.innerHTML = '<p class="muted">Loading records...</p>';
    if (scope === 'backoffice') {
        document.getElementById('record-hint').textContent = 'Viewing latest records';
    }

    try {
        const params = new URLSearchParams({ integration_id: integrationId });
        const response = await fetch(`${API_BASE_URL}/sync/records?${params.toString()}`);
        const data = await response.json();
        renderRecords(data.records || [], container);
    } catch (error) {
        console.error('Failed to load records', error);
        container.innerHTML = '<p class="muted">Failed to load records.</p>';
    }
}

function renderRecords(records, container) {
    if (!records.length) {
        container.innerHTML = '<p class="muted">No records found.</p>';
        return;
    }

    const table = document.createElement('table');
    table.className = 'records-table';
    table.innerHTML = `
        <thead>
            <tr>
                <th>Source ID</th>
                <th>Target ID</th>
                <th>Status</th>
                <th>Created</th>
                <th>Last Updated</th>
            </tr>
        </thead>
        <tbody>
            ${records.map(record => `
                <tr>
                    <td>${record.source_record_unique_id || record.source_record_id || '—'}</td>
                    <td>${record.destination_record_id || record.target_record_id || '—'}</td>
                    <td><span class="status-badge ${statusClass(record.status)}">${record.status || 'pending'}</span></td>
                    <td>${formatDate(record.created_at)}</td>
                    <td>${formatDate(record.updated_at)}</td>
                </tr>
            `).join('')}
        </tbody>
    `;
    container.innerHTML = '';
    container.appendChild(table);
}

function statusClass(status = '') {
    switch (status.toLowerCase()) {
        case 'synced': return 'status-synced';
        case 'failed': return 'status-failed';
        case 'pending': return 'status-pending';
        case 'retrying': return 'status-retrying';
        default: return 'status-pending';
    }
}

function refreshUserRecords() {
    if (!selectedIntegrationId) {
        showNotification('Select an integration first', 'info');
        return;
    }
    viewRecords(selectedIntegrationId, 'user');
}

function formatDate(value) {
    if (!value) return '—';
    try {
        return new Date(value).toLocaleString();
    } catch {
        return value;
    }
}

function showLoading() {
    document.getElementById('loading-overlay').classList.add('show');
}

function hideLoading() {
    document.getElementById('loading-overlay').classList.remove('show');
}

function showNotification(message, type = 'info') {
    console[type === 'error' ? 'error' : 'log'](message);
}

// Placeholder modals to keep UI responsive
function showAddIntegrationModal() {
    showNotification('Integration creation modal coming soon', 'info');
}

function showConnectGoogleModal() {
    showNotification('Google OAuth handled in dedicated flow', 'info');
}

function showConnectQuickBooksModal() {
    showNotification('QuickBooks OAuth coming soon', 'info');
}

function closeModal() {
    // placeholder
}
// API Configuration
const API_BASE_URL = 'http://localhost:8080/api';

// Global state
let currentCompany = null;
let currentPage = 1;
let currentFilters = {
    status: '',
    type: ''
};

// Initialize the application
document.addEventListener('DOMContentLoaded', function() {
    initializeApp();
});

function initializeApp() {
    setupEventListeners();
    loadCompanies();
}

function setupEventListeners() {
    // Navigation
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.addEventListener('click', function() {
            const tab = this.dataset.tab;
            switchTab(tab);
        });
    });

    // Filter change events
    document.getElementById('status-filter').addEventListener('change', function() {
        currentFilters.status = this.value;
    });

    document.getElementById('type-filter').addEventListener('change', function() {
        currentFilters.type = this.value;
    });

    // Form submissions
    document.getElementById('company-form').addEventListener('submit', handleCompanyForm);
    document.getElementById('integration-form').addEventListener('submit', handleIntegrationForm);
    document.getElementById('mapping-form').addEventListener('submit', handleMappingForm);
}

async function loadCompanies() {
    try {
        showLoading();
        
        // Load users (replacing companies in new architecture)
        const response = await fetch(`${API_BASE_URL}/users`);
        
        if (!response.ok) {
            const errorData = await response.json().catch(() => ({}));
            console.error('Error loading users:', errorData);
            
            // If backend is not available or database error, show helpful message
            if (response.status === 500 || response.status === 0) {
                showNotification('Backend server is not running or database is not connected. Please check your backend server.', 'error');
            } else {
                showNotification(errorData.error || 'Error loading users', 'error');
            }
            
            const select = document.getElementById('company-select');
            select.innerHTML = '<option value="">No users available</option>';
            return;
        }
        
        const data = await response.json();
        
        const select = document.getElementById('company-select');
        select.innerHTML = '<option value="">Select a user...</option>';
        
        if (data.users && data.users.length > 0) {
            data.users.forEach(user => {
                const option = document.createElement('option');
                option.value = user.id;
                option.textContent = user.company_name || user.email;
                select.appendChild(option);
            });
        } else {
            // No users found - show option to create one
            select.innerHTML = '<option value="">No users found. Please create a user first.</option>';
        }
        
    } catch (error) {
        console.error('Error loading users:', error);
        
        // Network error - backend might not be running
        if (error.message.includes('Failed to fetch') || error.message.includes('NetworkError')) {
            showNotification('Cannot connect to backend server. Make sure the backend is running on port 8080.', 'error');
        } else {
            showNotification('Error loading users: ' + error.message, 'error');
        }
        
        const select = document.getElementById('company-select');
        select.innerHTML = '<option value="">Connection error</option>';
    } finally {
        hideLoading();
    }
}

function switchCompany() {
    const select = document.getElementById('company-select');
    currentCompany = select.value;
    
    if (currentCompany) {
        loadDashboardData();
        showNotification(`Switched to company: ${select.selectedOptions[0].textContent}`, 'success');
    }
}

function switchTab(tabName) {
    // Remove active class from all tabs and buttons
    document.querySelectorAll('.tab-content').forEach(tab => {
        tab.classList.remove('active');
    });
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.classList.remove('active');
    });

    // Add active class to selected tab and button
    document.getElementById(tabName).classList.add('active');
    document.querySelector(`[data-tab="${tabName}"]`).classList.add('active');

    // Load data for the active tab
    if (tabName === 'dashboard' && currentCompany) {
        loadDashboardData();
    } else if (tabName === 'companies') {
        loadCompaniesTable();
    } else if (tabName === 'integrations' && currentCompany) {
        loadIntegrations();
    } else if (tabName === 'field-mappings' && currentCompany) {
        loadFieldMappings();
    } else if (tabName === 'records' && currentCompany) {
        loadRecords();
    }
}

async function loadDashboardData() {
    if (!currentCompany) {
        showNotification('Please select a company first', 'error');
        return;
    }

    try {
        showLoading();
        
        // Load statistics
        const statsResponse = await fetch(`${API_BASE_URL}/sync/stats?company_id=${currentCompany}`);
        const stats = await statsResponse.json();
        
        updateStatsDisplay(stats);
        loadRecentActivity();
        
    } catch (error) {
        console.error('Error loading dashboard data:', error);
        showNotification('Error loading dashboard data', 'error');
    } finally {
        hideLoading();
    }
}

function updateStatsDisplay(stats) {
    document.getElementById('total-records').textContent = stats.total_records || 0;
    document.getElementById('synced-records').textContent = stats.synced_records || 0;
    document.getElementById('failed-records').textContent = stats.failed_records || 0;
    document.getElementById('pending-records').textContent = stats.pending_records || 0;
}

function loadRecentActivity() {
    const activityList = document.getElementById('activity-list');
    activityList.innerHTML = `
        <div class="activity-item">
            <div class="activity-icon success">
                <i class="fas fa-check"></i>
            </div>
            <div>
                <strong>Account sync completed</strong>
                <p>5 accounts successfully synced to QuickBooks</p>
                <small>2 minutes ago</small>
            </div>
        </div>
        <div class="activity-item">
            <div class="activity-icon error">
                <i class="fas fa-exclamation"></i>
            </div>
            <div>
                <strong>Contact sync failed</strong>
                <p>Error: Invalid QuickBooks credentials</p>
                <small>5 minutes ago</small>
            </div>
        </div>
    `;
}

async function loadCompaniesTable() {
    try {
        showLoading();
        
        const response = await fetch(`${API_BASE_URL}/companies`);
        const data = await response.json();
        
        displayCompanies(data.companies);
        
    } catch (error) {
        console.error('Error loading companies:', error);
        showNotification('Error loading companies', 'error');
    } finally {
        hideLoading();
    }
}

function displayCompanies(companies) {
    const tbody = document.getElementById('companies-tbody');
    tbody.innerHTML = '';
    
    companies.forEach(company => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>${company.id}</td>
            <td>${company.name}</td>
            <td>${company.domain}</td>
            <td><span class="status-badge ${company.is_active ? 'status-synced' : 'status-failed'}">${company.is_active ? 'Active' : 'Inactive'}</span></td>
            <td>${formatDate(company.created_at)}</td>
            <td>
                <button class="btn btn-secondary" onclick="editCompany(${company.id})">
                    <i class="fas fa-edit"></i> Edit
                </button>
                <button class="btn btn-danger" onclick="deleteCompany(${company.id})">
                    <i class="fas fa-trash"></i> Delete
                </button>
            </td>
        `;
        tbody.appendChild(row);
    });
}

async function loadIntegrations() {
    if (!currentCompany) return;

    try {
        showLoading();
        
        // New API: /api/integrations?user_id=xxx
        const response = await fetch(`${API_BASE_URL}/integrations?user_id=${currentCompany}`);
        const data = await response.json();
        
        displayIntegrations(data.integrations || []);
        
    } catch (error) {
        console.error('Error loading integrations:', error);
        showNotification('Error loading integrations', 'error');
    } finally {
        hideLoading();
    }
}

async function loadConnections() {
    if (!currentCompany) return;
    
    try {
        // Load Google Drive connections
        const googleResponse = await fetch(`${API_BASE_URL}/connections?user_id=${currentCompany}&provider=google`);
        const googleData = await googleResponse.json();
        
        const sourceSelect = document.getElementById('source-connection-id');
        sourceSelect.innerHTML = '<option value="">Select Google Drive Connection</option>';
        if (googleData.connections) {
            googleData.connections.forEach(conn => {
                const option = document.createElement('option');
                option.value = conn.id;
                option.textContent = `Google Drive (${conn.provider_user_id || 'Connected'})`;
                sourceSelect.appendChild(option);
            });
        }
        
        // Load QuickBooks connections
        const qbResponse = await fetch(`${API_BASE_URL}/connections?user_id=${currentCompany}&provider=quickbooks`);
        const qbData = await qbResponse.json();
        
        const destSelect = document.getElementById('destination-connection-id');
        destSelect.innerHTML = '<option value="">Select QuickBooks Connection</option>';
        if (qbData.connections) {
            qbData.connections.forEach(conn => {
                const option = document.createElement('option');
                option.value = conn.id;
                option.textContent = `QuickBooks (${conn.realm_id || 'Connected'})`;
                destSelect.appendChild(option);
            });
        }
    } catch (error) {
        console.error('Error loading connections:', error);
    }
}

function displayIntegrations(integrations) {
    const grid = document.getElementById('integrations-grid');
    grid.innerHTML = '';
    
    if (integrations.length === 0) {
        grid.innerHTML = '<p>No integrations found. <button class="btn btn-primary" onclick="showAddIntegrationModal()">Add Integration</button></p>';
        return;
    }
    
    integrations.forEach(integration => {
        const card = document.createElement('div');
        card.className = 'integration-card';
        card.innerHTML = `
            <div class="integration-header">
                <h3>${integration.name}</h3>
                <span class="status-badge ${integration.status === 'active' ? 'status-synced' : 'status-failed'}">${integration.status || 'active'}</span>
            </div>
            <div class="integration-body">
                <p><strong>Source:</strong> Google Sheets</p>
                <p><strong>Destination:</strong> QuickBooks (${integration.destination_entity || 'N/A'})</p>
                <p><strong>Status:</strong> ${integration.status || 'active'}</p>
                <p><strong>Created:</strong> ${formatDate(integration.created_at)}</p>
            </div>
            <div class="integration-actions">
                <button class="btn btn-secondary" onclick="editIntegration(${integration.id})">
                    <i class="fas fa-edit"></i> Edit
                </button>
                <button class="btn btn-danger" onclick="deleteIntegration(${integration.id})">
                    <i class="fas fa-trash"></i> Delete
                </button>
            </div>
        `;
        grid.appendChild(card);
    });
}

async function loadFieldMappings() {
    if (!currentCompany) return;

    try {
        showLoading();
        
        const sourceSystem = document.getElementById('mapping-source-system').value;
        const targetSystem = document.getElementById('mapping-target-system').value;
        
        if (!sourceSystem || !targetSystem) {
            showNotification('Please select both source and target systems', 'error');
            return;
        }
        
        const response = await fetch(`${API_BASE_URL}/field-mappings?company_id=${currentCompany}&source_system=${sourceSystem}&target_system=${targetSystem}`);
        const data = await response.json();
        
        displayFieldMappings(data.mappings || []);
        
    } catch (error) {
        console.error('Error loading field mappings:', error);
        showNotification('Error loading field mappings', 'error');
    } finally {
        hideLoading();
    }
}

function displayFieldMappings(mappings) {
    const tbody = document.getElementById('field-mappings-tbody');
    tbody.innerHTML = '';
    
    if (mappings.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" style="text-align: center; padding: 20px;">No field mappings found</td></tr>';
        return;
    }
    
    mappings.forEach(mapping => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>${mapping.source_object}</td>
            <td>${mapping.source_field}</td>
            <td>${mapping.target_object}</td>
            <td>${mapping.target_field}</td>
            <td>${mapping.mapping_type}</td>
            <td>${mapping.is_required ? 'Yes' : 'No'}</td>
            <td>
                <button class="btn btn-secondary" onclick="editFieldMapping(${mapping.id})">
                    <i class="fas fa-edit"></i> Edit
                </button>
                <button class="btn btn-danger" onclick="deleteFieldMapping(${mapping.id})">
                    <i class="fas fa-trash"></i> Delete
                </button>
            </td>
        `;
        tbody.appendChild(row);
    });
}

async function loadRecords() {
    if (!currentCompany) {
        showNotification('Please select a company first', 'error');
        return;
    }

    try {
        showLoading();
        
        const params = new URLSearchParams({
            page: currentPage,
            limit: 10,
            company_id: currentCompany,
            ...currentFilters
        });
        
        const response = await fetch(`${API_BASE_URL}/sync/records?${params}`);
        const data = await response.json();
        
        displayRecords(data.records);
        updatePagination(data.pagination);
        
    } catch (error) {
        console.error('Error loading records:', error);
        showNotification('Error loading records', 'error');
    } finally {
        hideLoading();
    }
}

function displayRecords(records) {
    const tbody = document.getElementById('records-tbody');
    tbody.innerHTML = '';
    
    if (records.length === 0) {
        tbody.innerHTML = '<tr><td colspan="8" style="text-align: center; padding: 20px;">No records found</td></tr>';
        return;
    }
    
    records.forEach(record => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>${record.id}</td>
            <td>${record.company ? record.company.name : 'N/A'}</td>
            <td>${record.source_record_id}</td>
            <td>${record.target_record_id || 'N/A'}</td>
            <td>${record.record_type}</td>
            <td><span class="status-badge status-${record.status}">${record.status}</span></td>
            <td>${formatDate(record.last_sync_attempt || record.created_at)}</td>
            <td>
                <button class="btn btn-secondary" onclick="viewRecordDetails(${record.id})">
                    <i class="fas fa-eye"></i> View
                </button>
            </td>
        `;
        tbody.appendChild(row);
    });
}

function updatePagination(pagination) {
    const pageInfo = document.getElementById('page-info');
    pageInfo.textContent = `Page ${pagination.page} of ${pagination.pages}`;
    
    const prevBtn = document.getElementById('prev-page');
    const nextBtn = document.getElementById('next-page');
    
    prevBtn.disabled = pagination.page <= 1;
    nextBtn.disabled = pagination.page >= pagination.pages;
}

function changePage(direction) {
    const newPage = currentPage + direction;
    if (newPage >= 1) {
        currentPage = newPage;
        loadRecords();
    }
}

async function triggerSync(recordType) {
    if (!currentCompany) {
        showNotification('Please select a company first', 'error');
        return;
    }

    try {
        showLoading();
        
        const response = await fetch(`${API_BASE_URL}/sync/${currentCompany}/${recordType}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        });
        
        if (response.ok) {
            showNotification(`${recordType} sync triggered successfully`, 'success');
            loadDashboardData();
        } else {
            const error = await response.json();
            showNotification(`Error: ${error.details || 'Unknown error'}`, 'error');
        }
        
    } catch (error) {
        console.error('Error triggering sync:', error);
        showNotification('Error triggering sync', 'error');
    } finally {
        hideLoading();
    }
}

async function retryFailedSyncs() {
    if (!currentCompany) {
        showNotification('Please select a company first', 'error');
        return;
    }

    try {
        showLoading();
        
        const response = await fetch(`${API_BASE_URL}/sync/retry?company_id=${currentCompany}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        });
        
        if (response.ok) {
            showNotification('Retry process completed', 'success');
            loadDashboardData();
        } else {
            const error = await response.json();
            showNotification(`Error: ${error.details || 'Unknown error'}`, 'error');
        }
        
    } catch (error) {
        console.error('Error retrying syncs:', error);
        showNotification('Error retrying syncs', 'error');
    } finally {
        hideLoading();
    }
}

// Modal functions
function showAddCompanyModal() {
    document.getElementById('company-modal-title').textContent = 'Add Company';
    document.getElementById('company-form').reset();
    document.getElementById('company-modal').style.display = 'block';
}

function showAddIntegrationModal() {
    document.getElementById('integration-modal-title').textContent = 'Add Integration';
    document.getElementById('integration-form').reset();
    delete document.getElementById('integration-form').dataset.integrationId;
    
    // Load connections when opening modal
    loadConnections();
    
    document.getElementById('integration-modal').style.display = 'block';
}

async function showConnectGoogleModal() {
    if (!currentCompany) {
        showNotification('Please select a user first', 'error');
        return;
    }

    try {
        showLoading();
        
        // Get OAuth URL from backend
        const response = await fetch(`${API_BASE_URL}/oauth/google/initiate?user_id=${currentCompany}`);
        const data = await response.json();
        
        if (data.auth_url) {
            // Open OAuth window
            const width = 600;
            const height = 700;
            const left = (screen.width - width) / 2;
            const top = (screen.height - height) / 2;
            
            const oauthWindow = window.open(
                data.auth_url,
                'Google OAuth',
                `width=${width},height=${height},left=${left},top=${top}`
            );
            
            // Poll for window close (when OAuth completes)
            const checkClosed = setInterval(() => {
                if (oauthWindow.closed) {
                    clearInterval(checkClosed);
                    // Reload connections after OAuth completes
                    setTimeout(() => {
                        loadConnections();
                        showNotification('Google Drive connected successfully!', 'success');
                    }, 1000);
                }
            }, 500);
        } else {
            showNotification(data.error || 'Failed to initiate OAuth', 'error');
        }
    } catch (error) {
        console.error('Error initiating Google OAuth:', error);
        showNotification('Error connecting to Google Drive', 'error');
    } finally {
        hideLoading();
    }
}

function showConnectQuickBooksModal() {
    // TODO: Implement QuickBooks OAuth flow
    showNotification('QuickBooks OAuth connection will be implemented', 'info');
}

function showAddMappingModal() {
    document.getElementById('mapping-modal-title').textContent = 'Add Field Mapping';
    document.getElementById('mapping-form').reset();
    document.getElementById('mapping-modal').style.display = 'block';
}

function closeModal(modalId) {
    document.getElementById(modalId).style.display = 'none';
}

// Form handlers
async function handleCompanyForm(event) {
    event.preventDefault();
    
    const form = event.target;
    const companyId = form.dataset.companyId;
    const formData = {
        name: document.getElementById('company-name').value,
        domain: document.getElementById('company-domain').value,
        is_active: document.getElementById('company-active').checked
    };
    
    try {
        showLoading();
        
        const url = companyId ? `${API_BASE_URL}/companies/${companyId}` : `${API_BASE_URL}/companies`;
        const method = companyId ? 'PUT' : 'POST';
        
        const response = await fetch(url, {
            method: method,
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(formData)
        });
        
        if (response.ok) {
            showNotification(companyId ? 'Company updated successfully' : 'Company created successfully', 'success');
            closeModal('company-modal');
            delete form.dataset.companyId;
            loadCompanies();
            loadCompaniesTable();
        } else {
            const error = await response.json();
            showNotification(`Error: ${error.details || 'Unknown error'}`, 'error');
        }
        
    } catch (error) {
        console.error('Error saving company:', error);
        showNotification('Error saving company', 'error');
    } finally {
        hideLoading();
    }
}

async function handleIntegrationForm(event) {
    event.preventDefault();
    
    if (!currentCompany) {
        showNotification('Please select a company first', 'error');
        return;
    }
    
    const form = event.target;
    const integrationId = form.dataset.integrationId;
    
    // Build field mapping from CSV columns to QuickBooks fields
    // For now, we'll create a basic mapping structure
    const fieldMapping = {}; // This will be populated by the field mapping UI
    
    const formData = {
        user_id: currentCompany,
        name: document.getElementById('integration-name').value,
        source_connection_id: document.getElementById('source-connection-id').value,
        destination_connection_id: document.getElementById('destination-connection-id').value,
        source_file_id: document.getElementById('source-file-id').value,
        source_sheet_name: document.getElementById('source-sheet-name').value || '',
        source_primary_key: document.getElementById('source-primary-key').value,
        destination_entity: document.getElementById('destination-entity').value,
        field_mapping: JSON.stringify(fieldMapping),
        sync_frequency_minutes: parseInt(document.getElementById('sync-frequency').value) || 60,
        status: 'active'
    };
    
    try {
        showLoading();
        
        const url = integrationId 
            ? `${API_BASE_URL}/integrations/${integrationId}`
            : `${API_BASE_URL}/integrations`;
        const method = integrationId ? 'PUT' : 'POST';
        
        const response = await fetch(url, {
            method: method,
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(formData)
        });
        
        if (response.ok) {
            showNotification(integrationId ? 'Integration updated successfully' : 'Integration created successfully', 'success');
            closeModal('integration-modal');
            delete form.dataset.integrationId;
            loadIntegrations();
        } else {
            const error = await response.json();
            showNotification(`Error: ${error.details || 'Unknown error'}`, 'error');
        }
        
    } catch (error) {
        console.error('Error saving integration:', error);
        showNotification('Error saving integration', 'error');
    } finally {
        hideLoading();
    }
}

async function handleMappingForm(event) {
    event.preventDefault();
    
    if (!currentCompany) {
        showNotification('Please select a company first', 'error');
        return;
    }
    
    const form = event.target;
    const mappingId = form.dataset.mappingId;
    
    let transformRule = document.getElementById('mapping-transform-rule').value;
    if (transformRule) {
        try {
            // Try to parse as JSON to validate
            transformRule = JSON.parse(transformRule);
        } catch (e) {
            showNotification('Invalid JSON in transform rule field', 'error');
            return;
        }
    }
    
    const formData = {
        company_id: parseInt(currentCompany),
        source_system: document.getElementById('mapping-source-system').value,
        target_system: document.getElementById('mapping-target-system').value,
        source_object: document.getElementById('mapping-source-object').value,
        source_field: document.getElementById('mapping-source-field').value,
        target_object: document.getElementById('mapping-target-object').value,
        target_field: document.getElementById('mapping-target-field').value,
        mapping_type: document.getElementById('mapping-type').value,
        transform_rule: transformRule || null,
        is_required: document.getElementById('mapping-required').checked
    };
    
    try {
        showLoading();
        
        const url = mappingId ? `${API_BASE_URL}/field-mappings/${mappingId}` : `${API_BASE_URL}/field-mappings`;
        const method = mappingId ? 'PUT' : 'POST';
        
        const response = await fetch(url, {
            method: method,
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(formData)
        });
        
        if (response.ok) {
            showNotification(mappingId ? 'Field mapping updated successfully' : 'Field mapping created successfully', 'success');
            closeModal('mapping-modal');
            delete form.dataset.mappingId;
            loadFieldMappings();
        } else {
            const error = await response.json();
            showNotification(`Error: ${error.details || 'Unknown error'}`, 'error');
        }
        
    } catch (error) {
        console.error('Error saving field mapping:', error);
        showNotification('Error saving field mapping', 'error');
    } finally {
        hideLoading();
    }
}

// Edit and Delete functions
async function editCompany(id) {
    try {
        showLoading();
        
        const response = await fetch(`${API_BASE_URL}/companies/${id}`);
        const company = await response.json();
        
        document.getElementById('company-name').value = company.name;
        document.getElementById('company-domain').value = company.domain;
        document.getElementById('company-active').checked = company.is_active;
        
        document.getElementById('company-modal-title').textContent = 'Edit Company';
        document.getElementById('company-form').dataset.companyId = id;
        document.getElementById('company-modal').style.display = 'block';
        
    } catch (error) {
        console.error('Error loading company:', error);
        showNotification('Error loading company', 'error');
    } finally {
        hideLoading();
    }
}

async function deleteCompany(id) {
    if (!confirm('Are you sure you want to delete this company? This action cannot be undone.')) {
        return;
    }
    
    try {
        showLoading();
        
        const response = await fetch(`${API_BASE_URL}/companies/${id}`, {
            method: 'DELETE'
        });
        
        if (response.ok) {
            showNotification('Company deleted successfully', 'success');
            loadCompanies();
            loadCompaniesTable();
        } else {
            const error = await response.json();
            showNotification(`Error: ${error.details || 'Unknown error'}`, 'error');
        }
        
    } catch (error) {
        console.error('Error deleting company:', error);
        showNotification('Error deleting company', 'error');
    } finally {
        hideLoading();
    }
}

async function editIntegration(id) {
    try {
        showLoading();
        
        const response = await fetch(`${API_BASE_URL}/integrations/${id}`);
        const integration = await response.json();
        
        document.getElementById('integration-type').value = integration.type;
        document.getElementById('integration-name').value = integration.name;
        document.getElementById('integration-config').value = JSON.stringify(integration.config, null, 2);
        document.getElementById('integration-active').checked = integration.is_active;
        
        document.getElementById('integration-modal-title').textContent = 'Edit Integration';
        document.getElementById('integration-form').dataset.integrationId = id;
        document.getElementById('integration-modal').style.display = 'block';
        
    } catch (error) {
        console.error('Error loading integration:', error);
        showNotification('Error loading integration', 'error');
    } finally {
        hideLoading();
    }
}

async function deleteIntegration(id) {
    if (!confirm('Are you sure you want to delete this integration? This action cannot be undone.')) {
        return;
    }
    
    try {
        showLoading();
        
        const response = await fetch(`${API_BASE_URL}/integrations/${id}`, {
            method: 'DELETE'
        });
        
        if (response.ok) {
            showNotification('Integration deleted successfully', 'success');
            loadIntegrations();
        } else {
            const error = await response.json();
            showNotification(`Error: ${error.details || 'Unknown error'}`, 'error');
        }
        
    } catch (error) {
        console.error('Error deleting integration:', error);
        showNotification('Error deleting integration', 'error');
    } finally {
        hideLoading();
    }
}

async function editFieldMapping(id) {
    try {
        showLoading();
        
        const response = await fetch(`${API_BASE_URL}/field-mappings/${id}`);
        const mapping = await response.json();
        
        document.getElementById('mapping-source-system').value = mapping.source_system;
        document.getElementById('mapping-target-system').value = mapping.target_system;
        document.getElementById('mapping-source-object').value = mapping.source_object;
        document.getElementById('mapping-source-field').value = mapping.source_field;
        document.getElementById('mapping-target-object').value = mapping.target_object;
        document.getElementById('mapping-target-field').value = mapping.target_field;
        document.getElementById('mapping-type').value = mapping.mapping_type;
        document.getElementById('mapping-transform-rule').value = mapping.transform_rule ? JSON.stringify(mapping.transform_rule, null, 2) : '';
        document.getElementById('mapping-required').checked = mapping.is_required;
        
        document.getElementById('mapping-modal-title').textContent = 'Edit Field Mapping';
        document.getElementById('mapping-form').dataset.mappingId = id;
        document.getElementById('mapping-modal').style.display = 'block';
        
    } catch (error) {
        console.error('Error loading field mapping:', error);
        showNotification('Error loading field mapping', 'error');
    } finally {
        hideLoading();
    }
}

async function deleteFieldMapping(id) {
    if (!confirm('Are you sure you want to delete this field mapping? This action cannot be undone.')) {
        return;
    }
    
    try {
        showLoading();
        
        const response = await fetch(`${API_BASE_URL}/field-mappings/${id}`, {
            method: 'DELETE'
        });
        
        if (response.ok) {
            showNotification('Field mapping deleted successfully', 'success');
            loadFieldMappings();
        } else {
            const error = await response.json();
            showNotification(`Error: ${error.details || 'Unknown error'}`, 'error');
        }
        
    } catch (error) {
        console.error('Error deleting field mapping:', error);
        showNotification('Error deleting field mapping', 'error');
    } finally {
        hideLoading();
    }
}

async function viewRecordDetails(id) {
    try {
        showLoading();
        
        const response = await fetch(`${API_BASE_URL}/sync/records/${id}`);
        const record = await response.json();
        
        const detailsDiv = document.getElementById('record-details');
        detailsDiv.innerHTML = `
            <div class="record-details-content">
                <h3>Record Information</h3>
                <p><strong>ID:</strong> ${record.id}</p>
                <p><strong>Company:</strong> ${record.company ? record.company.name : 'N/A'}</p>
                <p><strong>Source Record ID:</strong> ${record.source_record_id}</p>
                <p><strong>Target Record ID:</strong> ${record.target_record_id || 'N/A'}</p>
                <p><strong>Record Type:</strong> ${record.record_type}</p>
                <p><strong>Status:</strong> <span class="status-badge status-${record.status}">${record.status}</span></p>
                <p><strong>Created:</strong> ${formatDate(record.created_at)}</p>
                <p><strong>Last Sync:</strong> ${formatDate(record.last_sync_attempt)}</p>
                ${record.error_message ? `<p><strong>Error:</strong> <span style="color: red;">${record.error_message}</span></p>` : ''}
                ${record.sync_data ? `<h4>Sync Data:</h4><pre>${JSON.stringify(record.sync_data, null, 2)}</pre>` : ''}
            </div>
        `;
        
        document.getElementById('record-modal').style.display = 'block';
        
    } catch (error) {
        console.error('Error loading record details:', error);
        showNotification('Error loading record details', 'error');
    } finally {
        hideLoading();
    }
}

// Utility functions
function showLoading() {
    document.getElementById('loading-overlay').classList.add('show');
}

function hideLoading() {
    document.getElementById('loading-overlay').classList.remove('show');
}

function showNotification(message, type = 'info') {
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.textContent = message;
    notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        padding: 15px 20px;
        border-radius: 5px;
        color: white;
        font-weight: 500;
        z-index: 1001;
        animation: slideIn 0.3s ease;
    `;
    
    if (type === 'success') {
        notification.style.background = '#28a745';
    } else if (type === 'error') {
        notification.style.background = '#dc3545';
    } else {
        notification.style.background = '#17a2b8';
    }
    
    document.body.appendChild(notification);
    
    setTimeout(() => {
        notification.remove();
    }, 3000);
}

function formatDate(dateString) {
    if (!dateString) return 'N/A';
    
    const date = new Date(dateString);
    return date.toLocaleString();
}

// Add CSS for notification animation
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from {
            transform: translateX(100%);
            opacity: 0;
        }
        to {
            transform: translateX(0);
            opacity: 1;
        }
    }
    
    .company-selector {
        background: rgba(255, 255, 255, 0.1);
        padding: 15px;
        border-radius: 10px;
        margin-bottom: 20px;
        display: flex;
        align-items: center;
        gap: 15px;
        backdrop-filter: blur(10px);
    }
    
    .company-selector label {
        color: white;
        font-weight: 500;
    }
    
    .company-selector select {
        padding: 8px 12px;
        border: none;
        border-radius: 5px;
        font-size: 1rem;
        min-width: 200px;
    }
    
    .integration-card {
        background: #f8f9fa;
        padding: 20px;
        border-radius: 10px;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        margin-bottom: 20px;
    }
    
    .integration-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 15px;
    }
    
    .integration-body p {
        margin: 5px 0;
        color: #666;
    }
    
    .integration-actions {
        margin-top: 15px;
        display: flex;
        gap: 10px;
    }
    
    .btn-danger {
        background: linear-gradient(135deg, #ff6b6b 0%, #ee5a52 100%);
        color: white;
    }
    
    .btn-danger:hover {
        transform: translateY(-2px);
        box-shadow: 0 5px 15px rgba(255, 107, 107, 0.4);
    }
`;
document.head.appendChild(style);
