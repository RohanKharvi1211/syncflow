const API_BASE_URL = 'http://localhost:8080/api';

let users = [];
let selectedBackofficeUserId = null;
let selectedIntegrationId = null;

document.addEventListener('DOMContentLoaded', () => {
    initializeApp();
});

async function initializeApp() {
    setupEventListeners();
    // Backoffice doesn't need to load users on init - will load after login
}

function setupEventListeners() {
    document.getElementById('login-form').addEventListener('submit', handleBackofficeLogin);
}

function handleBackofficeLogin(event) {
    event.preventDefault();
    const username = document.getElementById('username').value;
    const password = document.getElementById('password').value;
    
    // Simple authentication - in production, this should call an auth API
    if (username && password) {
        // For now, just proceed to dashboard
        // TODO: Implement proper authentication
        showDashboard();
        enterBackofficeView();
    } else {
        showNotification('Please enter username and password', 'error');
    }
}

function showLogin() {
    document.getElementById('login-screen').classList.remove('hidden');
    document.getElementById('dashboard').classList.add('hidden');
}

function showDashboard() {
    document.getElementById('login-screen').classList.add('hidden');
    document.getElementById('dashboard').classList.remove('hidden');
    document.getElementById('last-refresh-label').textContent = new Date().toLocaleString();
}

function logout() {
    selectedBackofficeUserId = null;
    selectedIntegrationId = null;
    showLogin();
    clearBackofficeView();
}

async function enterBackofficeView() {
    showLoading();
    try {
        await loadUsers();
        await loadCompanies();
    } catch (error) {
        console.error('Failed to load data', error);
        showNotification('Failed to load data', 'error');
    } finally {
        hideLoading();
    }
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

async function loadCompanies() {
    displayCompanies(users);
}

function displayCompanies(companies) {
    const container = document.getElementById('company-list');
    
    if (companies.length === 0) {
        container.innerHTML = '<p class="empty-state">No companies found</p>';
        return;
    }
    
    container.innerHTML = companies.map(user => `
        <div class="company-card ${selectedBackofficeUserId === user.id ? 'selected' : ''}" 
             onclick="selectCompany('${user.id}')">
            <h3>${user.company_name || user.email}</h3>
            <p class="muted">${user.email || 'No email'}</p>
        </div>
    `).join('');
}

async function selectCompany(userId) {
    selectedBackofficeUserId = userId;
    selectedIntegrationId = null;
    
    const user = users.find(u => u.id === userId);
    if (!user) return;
    
    document.getElementById('integration-hint').textContent = `Loading integrations for ${user.company_name || user.email}...`;
    document.getElementById('record-hint').textContent = 'Select an integration to view records';
    
    displayCompanies(users);
    
    showLoading();
    try {
        await loadBackofficeIntegrations(userId);
        document.getElementById('backoffice-records').innerHTML = '';
    } catch (error) {
        console.error('Error loading integrations:', error);
        showNotification('Error loading integrations', 'error');
    } finally {
        hideLoading();
    }
}

async function loadBackofficeIntegrations(userId) {
    if (!userId) {
        document.getElementById('backoffice-integrations').innerHTML = '<p class="empty-state">Select a company to view integrations</p>';
        return;
    }
    
    try {
        const response = await fetch(`${API_BASE_URL}/integrations?user_id=${userId}`);
        const data = await response.json();
        const integrations = data.integrations || [];
        
        displayBackofficeIntegrations(integrations);
    } catch (error) {
        console.error('Error loading integrations:', error);
        showNotification('Error loading integrations', 'error');
    }
}

function displayBackofficeIntegrations(integrations) {
    const container = document.getElementById('backoffice-integrations');
    
    if (integrations.length === 0) {
        container.innerHTML = '<p class="empty-state">No integrations found for this company</p>';
        document.getElementById('integration-hint').textContent = 'No integrations available';
        return;
    }
    
    container.innerHTML = integrations.map(integration => `
        <div class="integration-card ${selectedIntegrationId === integration.id ? 'selected' : ''}" 
             onclick="selectBackofficeIntegration('${integration.id}')">
            <div class="integration-header">
                <h3>${integration.name || 'Unnamed Integration'}</h3>
                <span class="status-badge ${integration.status === 'active' ? 'status-synced' : 'status-failed'}">
                    ${integration.status || 'active'}
                </span>
            </div>
            <div class="integration-body">
                <p><strong>Source:</strong> Google Sheets</p>
                <p><strong>Destination:</strong> QuickBooks (${integration.destination_entity || 'N/A'})</p>
                <p><strong>Created:</strong> ${formatDate(integration.created_at)}</p>
            </div>
        </div>
    `).join('');
    
    const user = users.find(u => u.id === selectedBackofficeUserId);
    document.getElementById('integration-hint').textContent = 
        `Showing ${integrations.length} integration(s) for ${user?.company_name || user?.email || 'company'}`;
}

async function selectBackofficeIntegration(integrationId) {
    selectedIntegrationId = integrationId;
    document.getElementById('record-hint').textContent = 'Loading records...';
    
    displayBackofficeIntegrations(
        await getIntegrationsForSelectedCompany()
    );
    
    showLoading();
    try {
        await loadBackofficeRecords(integrationId);
    } catch (error) {
        console.error('Error loading records:', error);
        showNotification('Error loading records', 'error');
    } finally {
        hideLoading();
    }
}

async function getIntegrationsForSelectedCompany() {
    if (!selectedBackofficeUserId) return [];
    
    try {
        const response = await fetch(`${API_BASE_URL}/integrations?user_id=${selectedBackofficeUserId}`);
        const data = await response.json();
        return data.integrations || [];
    } catch (error) {
        console.error('Error loading integrations:', error);
        return [];
    }
}

async function loadBackofficeRecords(integrationId) {
    if (!integrationId) {
        document.getElementById('backoffice-records').innerHTML = '<p class="empty-state">Select an integration to view records</p>';
        return;
    }
    
    try {
        const response = await fetch(`${API_BASE_URL}/sync/records?integration_id=${integrationId}`);
        const data = await response.json();
        const records = data.records || [];
        
        displayBackofficeRecords(records);
    } catch (error) {
        console.error('Error loading records:', error);
        showNotification('Error loading records', 'error');
    }
}

function displayBackofficeRecords(records) {
    const container = document.getElementById('backoffice-records');
    
    if (records.length === 0) {
        container.innerHTML = '<p class="empty-state">No sync records found for this integration.</p>';
        document.getElementById('record-hint').textContent = 'No records available';
        return;
    }
    
    const table = `
        <table class="records-table">
            <thead>
                <tr>
                    <th>Source ID</th>
                    <th>Destination ID</th>
                    <th>Status</th>
                    <th>Last Sync</th>
                    <th>Error</th>
                </tr>
            </thead>
            <tbody>
                ${records.map(record => `
                    <tr>
                        <td>${record.source_record_unique_id || '-'}</td>
                        <td>${record.destination_record_id || '-'}</td>
                        <td>
                            <span class="status-badge ${getStatusClass(record.status)}">
                                ${record.status || 'pending'}
                            </span>
                        </td>
                        <td>${formatDate(record.updated_at || record.created_at)}</td>
                        <td>${record.error_message ? `<span class="error-text">${record.error_message}</span>` : '-'}</td>
                    </tr>
                `).join('')}
            </tbody>
        </table>
    `;
    
    container.innerHTML = table;
    document.getElementById('record-hint').textContent = `Showing ${records.length} record(s) - ${records.filter(r => r.status === 'synced').length} synced, ${records.filter(r => r.status === 'failed').length} failed`;
}

function getStatusClass(status) {
    const statusMap = {
        'synced': 'status-synced',
        'failed': 'status-failed',
        'pending': 'status-pending',
        'retrying': 'status-retrying'
    };
    return statusMap[status] || 'status-pending';
}

function clearBackofficeView() {
    document.getElementById('company-list').innerHTML = '';
    document.getElementById('backoffice-integrations').innerHTML = '';
    document.getElementById('backoffice-records').innerHTML = '';
    document.getElementById('integration-hint').textContent = 'Select a company to view integrations';
    document.getElementById('record-hint').textContent = 'Select an integration to view sync records';
}

function formatDate(dateString) {
    if (!dateString) return '-';
    const date = new Date(dateString);
    return date.toLocaleString();
}

function showLoading() {
    document.getElementById('loading-overlay').classList.remove('hidden');
}

function hideLoading() {
    document.getElementById('loading-overlay').classList.add('hidden');
}

function showNotification(message, type = 'info') {
    // Simple notification - you can enhance this with a toast library
    alert(message);
}





