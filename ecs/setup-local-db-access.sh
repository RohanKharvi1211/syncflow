#!/bin/bash
# Script to set up local access to RDS database
# This allows your IP and tests the connection

set -e

AWS_REGION=${AWS_REGION:-ap-south-1}
RDS_SG_ID="sg-0d156ead6b2f3fa37"

echo "=========================================="
echo "Setting Up Local Access to RDS"
echo "=========================================="
echo ""

# Get your public IP (try both IPv4 and IPv6)
MY_IPV4=$(curl -s -4 ifconfig.me 2>/dev/null || curl -s -4 https://api.ipify.org 2>/dev/null || echo "")
MY_IPV6=$(curl -s -6 ifconfig.me 2>/dev/null || curl -s -6 https://api6.ipify.org 2>/dev/null || echo "")

echo "Detected IP addresses:"
if [ ! -z "$MY_IPV4" ]; then
    echo "  IPv4: $MY_IPV4"
fi
if [ ! -z "$MY_IPV6" ]; then
    echo "  IPv6: $MY_IPV6"
fi
echo ""

if [ -z "$MY_IPV4" ] && [ -z "$MY_IPV6" ]; then
    echo "❌ Error: Could not detect your public IP"
    echo "   Please visit https://ifconfig.me to get your IP manually"
    exit 1
fi

# Allow IPv4 if available
if [ ! -z "$MY_IPV4" ]; then
    echo "Allowing IPv4 access: $MY_IPV4/32"
    aws ec2 authorize-security-group-ingress \
        --group-id "$RDS_SG_ID" \
        --protocol tcp \
        --port 5432 \
        --cidr "${MY_IPV4}/32" \
        --region $AWS_REGION 2>&1 | grep -v "already exists" || echo "  ✓ IPv4 rule already exists or added"
fi

# Allow IPv6 if available
if [ ! -z "$MY_IPV6" ]; then
    echo "Allowing IPv6 access: $MY_IPV6/128"
    aws ec2 authorize-security-group-ingress \
        --group-id "$RDS_SG_ID" \
        --ip-permissions IpProtocol=tcp,FromPort=5432,ToPort=5432,Ipv6Ranges=[{CidrIpv6=${MY_IPV6}/128}] \
        --region $AWS_REGION 2>&1 | grep -v "already exists" || echo "  ✓ IPv6 rule already exists or added"
fi

echo ""
echo "=========================================="
echo "✓ Security Group Updated!"
echo "=========================================="
echo ""

# Check if psql is installed
if ! command -v psql &> /dev/null; then
    echo "⚠️  PostgreSQL client (psql) is not installed."
    echo ""
    echo "Install it with:"
    echo "  macOS:   brew install postgresql"
    echo "  Ubuntu:  sudo apt-get install postgresql-client"
    echo ""
    echo "After installing, you can connect using:"
    echo "  cd ecs && ./connect-to-rds.sh"
    exit 0
fi

echo "Testing database connection..."
echo ""

# Get database credentials
DB_HOST=$(aws rds describe-db-instances \
    --db-instance-identifier syncflow-db \
    --region $AWS_REGION \
    --query 'DBInstances[0].Endpoint.Address' \
    --output text 2>/dev/null)

DB_PASSWORD=$(aws secretsmanager get-secret-value \
    --secret-id syncflow/database/password \
    --region $AWS_REGION \
    --query 'SecretString' \
    --output text 2>/dev/null)

if [ -z "$DB_HOST" ] || [ -z "$DB_PASSWORD" ]; then
    echo "⚠️  Could not get database credentials. You can connect manually using:"
    echo "   cd ecs && ./connect-to-rds.sh"
    exit 0
fi

# Test connection
echo "Attempting to connect to: $DB_HOST"
if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p 5432 -U postgres -d postgres -c "SELECT version();" --set=sslmode=require >/dev/null 2>&1; then
    echo "✅ Connection successful!"
    echo ""
    echo "You can now connect using:"
    echo "  cd ecs && ./connect-to-rds.sh"
    echo ""
    echo "Or directly:"
    echo "  PGPASSWORD='$DB_PASSWORD' psql -h $DB_HOST -p 5432 -U postgres -d postgres --set=sslmode=require"
else
    echo "⚠️  Connection test failed. This could be:"
    echo "   - Security group rules haven't propagated yet (wait 30 seconds)"
    echo "   - Network connectivity issue"
    echo "   - SSL configuration issue"
    echo ""
    echo "You can still try connecting manually:"
    echo "  cd ecs && ./connect-to-rds.sh"
fi

