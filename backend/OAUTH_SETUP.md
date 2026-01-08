# Google Drive OAuth Setup

## Overview

The Google Drive OAuth flow has been implemented. When users click "Connect Google Drive", they will be redirected to Google's OAuth consent screen.

## Setup Instructions

### 1. Create Google OAuth Credentials

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the **Google Drive API**
4. Go to **Credentials** → **Create Credentials** → **OAuth 2.0 Client ID**
5. Configure:
   - Application type: **Web application**
   - Authorized redirect URIs: `http://localhost:8080/api/oauth/google/callback`
   - For production: Add your production callback URL

### 2. Set Environment Variables

Add these to your `.env` file or environment:

```env
GOOGLE_CLIENT_ID=your_google_client_id_here
GOOGLE_CLIENT_SECRET=your_google_client_secret_here
GOOGLE_REDIRECT_URI=http://localhost:8080/api/oauth/google/callback
```

### 3. How It Works

1. **User clicks "Connect Google Drive"** in the frontend
2. **Frontend calls** `/api/oauth/google/initiate?user_id=xxx`
3. **Backend returns** OAuth URL
4. **Frontend opens** OAuth URL in popup window
5. **User authorizes** on Google's consent screen
6. **Google redirects** to `/api/oauth/google/callback?code=xxx&state=xxx`
7. **Backend exchanges** code for access/refresh tokens
8. **Backend saves** tokens to `connections` table
9. **Backend redirects** to success page
10. **Frontend detects** popup close and refreshes connections list

## API Endpoints

- `GET /api/oauth/google/initiate?user_id=xxx` - Get OAuth URL
- `GET /api/oauth/google/callback?code=xxx&state=xxx` - Handle OAuth callback

## Security Notes

- The `state` parameter includes the user_id for security
- Tokens are stored in the database (should be encrypted in production)
- Refresh tokens are stored for automatic token renewal
- Access tokens expire after 1 hour and need to be refreshed

## Next Steps

1. Set up your Google OAuth credentials
2. Add the environment variables
3. Test the OAuth flow
4. Implement token refresh logic (when tokens expire)





