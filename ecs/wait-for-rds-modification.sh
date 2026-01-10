#!/bin/bash
# Script to wait for RDS modification to complete
# Usage: ./wait-for-rds-modification.sh

AWS_REGION=${AWS_REGION:-ap-south-1}
DB_INSTANCE_ID="syncflow-db"

echo "=========================================="
echo "Waiting for RDS Modification to Complete"
echo "=========================================="
echo "This usually takes 5-10 minutes..."
echo ""

TIMEOUT=600  # 10 minutes
ELAPSED=0
CHECK_INTERVAL=30  # Check every 30 seconds

while [ $ELAPSED -lt $TIMEOUT ]; do
    STATUS=$(aws rds describe-db-instances \
        --db-instance-identifier "$DB_INSTANCE_ID" \
        --region $AWS_REGION \
        --query 'DBInstances[0].DBInstanceStatus' \
        --output text 2>/dev/null || echo "UNKNOWN")
    
    PUBLICLY_ACCESSIBLE=$(aws rds describe-db-instances \
        --db-instance-identifier "$DB_INSTANCE_ID" \
        --region $AWS_REGION \
        --query 'DBInstances[0].PubliclyAccessible' \
        --output text 2>/dev/null || echo "UNKNOWN")
    
    echo "[${ELAPSED}s] Status: $STATUS | Publicly Accessible: $PUBLICLY_ACCESSIBLE"
    
    if [ "$STATUS" == "available" ] && [ "$PUBLICLY_ACCESSIBLE" == "True" ]; then
        echo ""
        echo "=========================================="
        echo "✅ RDS Modification Complete!"
        echo "=========================================="
        echo "RDS is now publicly accessible."
        echo ""
        echo "You can now connect using:"
        echo "  cd ecs && ./connect-to-rds.sh"
        echo ""
        echo "Or via DBeaver:"
        echo "  Host: syncflow-db.cvom0au8yuea.ap-south-1.rds.amazonaws.com"
        echo "  Port: 5432"
        echo "  Database: postgres"
        echo "  Username: postgres"
        echo "  Password: (get from Secrets Manager)"
        echo "  SSL Mode: require"
        exit 0
    fi
    
    sleep $CHECK_INTERVAL
    ELAPSED=$((ELAPSED + CHECK_INTERVAL))
done

echo ""
echo "⚠️  Timeout waiting for RDS modification to complete."
echo "   Please check manually:"
echo "   aws rds describe-db-instances --db-instance-identifier syncflow-db --region ap-south-1 --query 'DBInstances[0].{Status:DBInstanceStatus,PubliclyAccessible:PubliclyAccessible}' --output table"
exit 1

