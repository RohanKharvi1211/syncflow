const API_BASE_URL = 'http://localhost:8080/api';

let users = [];
let currentUser = null;
let selectedIntegrationId = null;
let companyPreview = '';

document.addEventListener('DOMContentLoaded', () => {
    initializeApp();
});

async function initializeApp() {
    setupEventListeners();
    
    // Check if user is already logged in (from localStorage) FIRST
    const savedUser = localStorage.getItem('currentUser');
    if (savedUser) {
        try {
            currentUser = JSON.parse(savedUser);
            await loadUserData();
            showDashboard();
            return; // Don't show login if user is already logged in
        } catch (error) {
            console.error('Error loading saved user:', error);
            localStorage.removeItem('currentUser');
            currentUser = null;
        }
    }
    
    // Only show login if no saved user found
    showLogin();
}

function setupEventListeners() {
    document.getElementById('login-form').addEventListener('submit', handleLogin);
    document.getElementById('email-input').addEventListener('input', previewCompanyFromEmail);
}

async function loadUsers() {
    // Only load users if we have a company_id (user is logged in)
    if (!currentUser || !currentUser.company_id) {
        return;
    }
    
    try {
        const response = await fetch(`${API_BASE_URL}/users?company_id=${currentUser.company_id}`);
        if (response.ok) {
            const data = await response.json();
            users = data.users || [];
        }
    } catch (error) {
        console.error('Failed to load users', error);
        // Don't show error for unauthenticated users
    }
}

async function loadUserData() {
    // Load user-specific data after login
    await loadUsers();
    // Load integrations, connections, etc.
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

function handleLogin(event) {
    event.preventDefault();
    
    const emailInput = document.getElementById('email-input');
    const email = emailInput.value.trim();
    
    if (!email) {
        showNotification('Please enter your work email', 'error');
        return;
    }
    
    if (!validateEmail(email)) {
        showNotification('Please enter a valid work email (e.g., rohan@flipkart.com)', 'error');
        return;
    }
    
    const emailLower = email.toLowerCase();
    
    const detectedCompany = extractCompanyFromEmail(emailLower);
    document.getElementById('company-preview').value = detectedCompany || '';
    
    const matchedUser = findUserByEmail(emailLower, detectedCompany);
    if (!matchedUser) {
        showNotification(`We couldn't find a company linked to ${detectedCompany || 'that email'}. Please contact support.`, 'error');
        return;
    }
    
    currentUser = matchedUser;
    // Save user to localStorage for persistent login
    localStorage.setItem('currentUser', JSON.stringify(currentUser));
    document.getElementById('current-user-label').textContent = currentUser.company_name || detectedCompany || email;
    enterUserView();
    showDashboard();
}

function logout() {
    currentUser = null;
    selectedIntegrationId = null;
    // Clear saved user from localStorage
    localStorage.removeItem('currentUser');
    document.getElementById('email-input').value = '';
    document.getElementById('company-preview').value = '';
    showLogin();
    clearUserView();
}

async function signInWithGoogle() {
    try {
        showLoading();
        
        // Generate state for security (can include user_id or session info)
        const state = generateState();
        
        // Build OAuth URL directly in frontend (Client ID is public)
        const clientId = window.FRONTEND_CONFIG?.GOOGLE_CLIENT_ID || 'YOUR_GOOGLE_CLIENT_ID_HERE';
        const redirectUri = window.FRONTEND_CONFIG?.GOOGLE_REDIRECT_URI || 'http://localhost:8080/api/oauth/google/callback';
        const scopes = window.FRONTEND_CONFIG?.GOOGLE_SCOPES || 'https://www.googleapis.com/auth/drive.readonly';
        
        if (clientId === 'YOUR_GOOGLE_CLIENT_ID_HERE') {
            showNotification('Google OAuth not configured. Please set GOOGLE_CLIENT_ID in config.js', 'error');
            hideLoading();
            return;
        }
        
        // Build OAuth URL
        const authUrl = `https://accounts.google.com/o/oauth2/v2/auth?` +
            `client_id=${encodeURIComponent(clientId)}&` +
            `redirect_uri=${encodeURIComponent(redirectUri)}&` +
            `response_type=code&` +
            `scope=${encodeURIComponent(scopes)}&` +
            `access_type=offline&` +
            `prompt=consent&` +
            `state=${encodeURIComponent(state)}`;
        
        hideLoading();
        
        // Open OAuth window
        const width = 600;
        const height = 700;
        const left = (screen.width - width) / 2;
        const top = (screen.height - height) / 2;
        
        const oauthWindow = window.open(
            authUrl,
            'Google OAuth',
            `width=${width},height=${height},left=${left},top=${top}`
        );
        
        // Listen for OAuth callback message from popup
        window.addEventListener('message', handleOAuthCallback);
        
        // Poll for window close
        const checkClosed = setInterval(() => {
            if (oauthWindow.closed) {
                clearInterval(checkClosed);
                window.removeEventListener('message', handleOAuthCallback);
            }
        }, 500);
    } catch (error) {
        console.error('Error initiating Google OAuth:', error);
        showNotification('Error connecting to Google. Please use email sign-in instead.', 'error');
        hideLoading();
    }
}

// Generate state parameter for OAuth security
function generateState() {
    // Can include user_id, session token, or random string
    // Backend will validate this in the callback
    return `temp|${Date.now()}-${Math.random().toString(36).substring(7)}`;
}

async function handleOAuthCallback(event) {
    // Handle message from OAuth callback page
    if (event.data && event.data.type === 'oauth-success') {
        const email = event.data.email;
        if (email && validateEmail(email)) {
            showLoading();
            try {
                // Find user by email via API
                const response = await fetch(`${API_BASE_URL}/users?email=${encodeURIComponent(email)}`);
                if (response.ok) {
                    const data = await response.json();
                    const matchedUser = data.user;
                    
                    if (matchedUser) {
                        currentUser = matchedUser;
                        // Save user to localStorage for persistent login
                        localStorage.setItem('currentUser', JSON.stringify(currentUser));
                        document.getElementById('current-user-label').textContent = matchedUser.company_name || email;
                        
                        // Load users list for this company
                        await loadUsers();
                        
                        enterUserView();
                        showDashboard();
                        showNotification('Signed in successfully with Google!', 'success');
                    } else {
                        showNotification(`We couldn't find a user account for ${email}. Please contact support.`, 'error');
                    }
                } else if (response.status === 404) {
                    showNotification(`We couldn't find a user account for ${email}. Please contact support.`, 'error');
                } else {
                    const errorData = await response.json().catch(() => ({}));
                    showNotification(errorData.error || 'Failed to find user account. Please try email sign-in.', 'error');
                }
            } catch (error) {
                console.error('Error finding user:', error);
                showNotification('Failed to find user account. Please try email sign-in.', 'error');
            } finally {
                hideLoading();
            }
        }
    } else if (event.data && event.data.type === 'oauth-error') {
        showNotification(event.data.error || 'Google sign-in failed. Please try email sign-in.', 'error');
    }
}

function previewCompanyFromEmail() {
    const email = document.getElementById('email-input').value.trim().toLowerCase();
    const company = extractCompanyFromEmail(email);
    companyPreview = company;
    document.getElementById('company-preview').value = company || '';
}

function extractCompanyFromEmail(email) {
    if (!email.includes('@')) return '';
    const domain = email.split('@')[1] || '';
    if (!domain.includes('.')) return '';
    const slug = domain.split('.')[0];
    if (!slug) return '';
    return slug.charAt(0).toUpperCase() + slug.slice(1);
}

function validateEmail(email) {
    if (!email || typeof email !== 'string') return false;
    // Simple email validation: has @ and at least one dot after @
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email.trim());
}

function findUserByEmail(email, detectedCompany) {
    const domain = (email.split('@')[1] || '').toLowerCase();
    const normalizedCompany = (detectedCompany || '').toLowerCase();
    
    return users.find(user => {
        const companyName = (user.company_name || '').toLowerCase();
        const userEmail = (user.email || '').toLowerCase();
        return userEmail === email ||
            userEmail.endsWith(`@${domain}`) ||
            companyName === normalizedCompany ||
            companyName.includes(normalizedCompany);
    });
}

async function enterUserView() {
    if (!currentUser) return;
    
    showLoading();
    try {
        await loadUserIntegrations();
    } catch (error) {
        console.error('Failed to load integrations', error);
        showNotification('Failed to load integrations', 'error');
    } finally {
        hideLoading();
    }
}

async function loadUserIntegrations() {
    if (!currentUser) return;
    
    try {
        const response = await fetch(`${API_BASE_URL}/integrations?user_id=${currentUser.id}`);
        const data = await response.json();
        const integrations = data.integrations || [];
        
        displayUserIntegrations(integrations);
    } catch (error) {
        console.error('Error loading integrations:', error);
        showNotification('Error loading integrations', 'error');
    }
}

function displayUserIntegrations(integrations) {
    const container = document.getElementById('user-integrations');
    
    if (integrations.length === 0) {
        container.innerHTML = '<p class="empty-state">No integrations found. Contact support to set up your first integration.</p>';
        return;
    }
    
    container.innerHTML = integrations.map(integration => `
        <div class="integration-card" onclick="selectIntegration('${integration.id}')">
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
}

async function selectIntegration(integrationId) {
    selectedIntegrationId = integrationId;
    document.getElementById('record-hint').textContent = 'Loading records...';
    
    try {
        await loadUserRecords(integrationId);
    } catch (error) {
        console.error('Error loading records:', error);
        showNotification('Error loading records', 'error');
    }
}

async function loadUserRecords(integrationId) {
    if (!integrationId) return;
    
    try {
        const response = await fetch(`${API_BASE_URL}/sync/records?integration_id=${integrationId}`);
        const data = await response.json();
        const records = data.records || [];
        
        displayUserRecords(records);
    } catch (error) {
        console.error('Error loading records:', error);
        showNotification('Error loading records', 'error');
    }
}

function displayUserRecords(records) {
    const container = document.getElementById('user-records');
    
    if (records.length === 0) {
        container.innerHTML = '<p class="empty-state">No sync records found for this integration.</p>';
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
                    <th>Actions</th>
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
                        <td>
                            <button class="btn btn-sm" onclick="viewRecordDetails('${record.id}')">
                                View
                            </button>
                        </td>
                    </tr>
                `).join('')}
            </tbody>
        </table>
    `;
    
    container.innerHTML = table;
    document.getElementById('record-hint').textContent = `Showing ${records.length} record(s)`;
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

async function viewRecordDetails(recordId) {
    try {
        const response = await fetch(`${API_BASE_URL}/sync/records/${recordId}`);
        const record = await response.json();
        
        const details = `
            <div class="record-details">
                <h3>Record Details</h3>
                <p><strong>Status:</strong> <span class="status-badge ${getStatusClass(record.status)}">${record.status}</span></p>
                <p><strong>Source ID:</strong> ${record.source_record_unique_id || '-'}</p>
                <p><strong>Destination ID:</strong> ${record.destination_record_id || '-'}</p>
                <p><strong>Created:</strong> ${formatDate(record.created_at)}</p>
                <p><strong>Updated:</strong> ${formatDate(record.updated_at)}</p>
                ${record.error_message ? `<p><strong>Error:</strong> ${record.error_message}</p>` : ''}
            </div>
        `;
        
        document.getElementById('integration-details').innerHTML = details;
        document.getElementById('integration-modal-title').textContent = 'Record Details';
        document.getElementById('integration-modal').classList.remove('hidden');
    } catch (error) {
        console.error('Error loading record details:', error);
        showNotification('Error loading record details', 'error');
    }
}

function clearUserView() {
    document.getElementById('user-integrations').innerHTML = '';
    document.getElementById('user-records').innerHTML = '';
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

function closeModal(modalId) {
    document.getElementById(modalId).classList.add('hidden');
}

