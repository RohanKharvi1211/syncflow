# Bot Protection Added

## Problem
You were seeing repeated 404 errors in logs from automated bot scanners:
```
[GIN] 2026/01/10 - 14:00:34 | 404 | 135.673µs | 158.220.90.195 | GET "/config/keys/index.php"
```

**IP: 158.220.90.195** is a bot/scanner trying to find common vulnerabilities by probing for:
- Exposed configuration files (`/config/keys/index.php`)
- Admin panels
- Vulnerable PHP applications
- Security misconfigurations

## Solution Implemented

I've added two middlewares to your backend:

### 1. Bot Protection Middleware
- **Blocks common scanner paths** before they reach your application
- **Reduces log noise** by returning 404 immediately
- **Prevents unnecessary resource usage** on legitimate requests

**Blocked paths include:**
- `/config*` (like `/config/keys/index.php`)
- `/.env`, `/.git`
- `/wp-admin`, `/wp-login`, `/phpmyadmin`
- `/admin`, `/administrator`
- `/.well-known`, `/phpinfo`, `/shell`, `/cgi-bin`
- `*.php`, `*.jsp`, `*.asp` (suspicious file extensions)

### 2. Rate Limiting Middleware
- **Limits requests per IP**: 60 requests per minute (configurable)
- **Prevents abuse** and DDoS attempts
- **In-memory tracking** (simple but effective)

**Default settings:**
- Rate: 60 requests per minute per IP
- Window: 1 minute
- Response: HTTP 429 (Too Many Requests) when exceeded

## Impact

### Before:
- ❌ Every bot request logs a 404 error
- ❌ Logs are cluttered with scanner attempts
- ❌ Wastes resources processing invalid requests
- ❌ No protection against abuse

### After:
- ✅ Bot requests are blocked early (no log noise)
- ✅ Cleaner logs with only legitimate requests
- ✅ Reduced resource usage
- ✅ Rate limiting prevents abuse
- ✅ Better security posture

## Files Changed

1. **`backend/cmd/app/middlewares/bot_protection.go`** (new)
   - Blocks common scanner paths
   - Returns 404 immediately for suspicious requests

2. **`backend/cmd/app/middlewares/rate_limit.go`** (new)
   - In-memory rate limiting
   - 60 requests/minute per IP (configurable)

3. **`backend/cmd/app/middlewares/middlewares.go`** (updated)
   - Added BotProtection and RateLimit to Middlewares struct

4. **`backend/cmd/app/app.go`** (updated)
   - Added middlewares in correct order:
     1. BotProtection (first - blocks early)
     2. RateLimit (second - prevents abuse)
     3. CORS (third - allows cross-origin)
     4. Logger (last - logs after filtering)

## Deployment

To deploy this fix:

```bash
git add backend/cmd/app/middlewares/bot_protection.go \
        backend/cmd/app/middlewares/rate_limit.go \
        backend/cmd/app/middlewares/middlewares.go \
        backend/cmd/app/app.go

git commit -m "Add bot protection and rate limiting middleware"
git push origin main
```

After deployment (2-3 minutes), bot requests will be blocked silently without logging 404 errors.

## Configuration

### Adjust Rate Limit (if needed)

Edit `backend/cmd/app/middlewares/rate_limit.go`:

```go
func init() {
    // Change these values:
    globalRateLimiter = NewRateLimiter(
        60,           // requests per minute
        time.Minute,  // time window
    )
}
```

### Add More Blocked Paths (if needed)

Edit `backend/cmd/app/middlewares/bot_protection.go`:

```go
blockedPaths := []string{
    "/config",
    "/.env",
    // Add more paths here:
    "/new-suspicious-path",
}
```

## Testing

After deployment, you can test:

1. **Bot protection** - Try accessing `/config/keys/index.php`:
   ```bash
   curl https://your-alb-dns/config/keys/index.php
   # Should return: 404 (silently, no log entry)
   ```

2. **Rate limiting** - Make more than 60 requests in a minute:
   ```bash
   for i in {1..65}; do curl https://your-alb-dns/health; done
   # After 60 requests, should return: 429 Too Many Requests
   ```

## Additional Security Recommendations

### 1. AWS WAF (Web Application Firewall)
For production, consider adding AWS WAF to the ALB:
- Managed rule sets for common attacks
- IP reputation lists
- Geographic blocking
- Custom rules

**Cost:** ~$5/month + $1 per million requests

### 2. CloudWatch Alarms
Set up alerts for unusual activity:
- High 404 rate (indicates scanner)
- High 429 rate (indicates attack)
- Unusual traffic patterns

### 3. Security Group Rules (Optional)
You could restrict ALB access, but this might block legitimate users:
```bash
# Not recommended - too restrictive for web application
# ALB needs to be public (0.0.0.0/0) for web access
```

### 4. Log Filtering
Already handled by BotProtection middleware, but you can also filter in CloudWatch:
- Filter out 404s from specific IPs
- Alert on repeated patterns

## Summary

✅ **Bot protection middleware added** - Blocks common scanner paths  
✅ **Rate limiting middleware added** - Prevents abuse  
✅ **Log noise reduced** - No more 404 spam in logs  
✅ **Better security** - Early blocking of suspicious requests  
✅ **No breaking changes** - All existing functionality preserved  

**Next step:** Deploy the changes and the bot requests will be silently blocked!

