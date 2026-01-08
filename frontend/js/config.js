// Frontend Configuration
// Client IDs can be public - they're meant to be exposed in frontend code
// Client SECRETS must stay in backend only

const FRONTEND_CONFIG = {
    // Google OAuth Client ID (public - safe to expose)
    // Get this from: https://console.cloud.google.com/
    GOOGLE_CLIENT_ID: '485007484288-04itsc51bn5lgbam7qk839kkvduj6tkb.apps.googleusercontent.com',
    
    // OAuth Redirect URI
    GOOGLE_REDIRECT_URI: 'http://localhost:8080/api/oauth/google/callback',
    
    // API Base URL
    API_BASE_URL: 'http://localhost:8080/api',
    
    // OAuth Scopes - include userinfo for getting user email/name
    GOOGLE_SCOPES: 'https://www.googleapis.com/auth/drive.readonly https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile'
};

// Make it available globally
window.FRONTEND_CONFIG = FRONTEND_CONFIG;

