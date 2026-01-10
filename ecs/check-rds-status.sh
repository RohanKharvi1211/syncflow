#!/bin/bash
# Quick script to check RDS status
# Usage: ./check-rds-status.sh

AWS_REGION=${AWS_REGION:-ap-south-1}

echo "=========================================="
echo "RDS Status Check"
echo "=========================================="
echo ""

aws rds describe-db-instances \
    --db-instance-identifier syncflow-db \
    --region $AWS_REGION \
    --query 'DBInstances[0].{
        Status:DBInstanceStatus,
        PubliclyAccessible:PubliclyAccessible,
        Endpoint:Endpoint.Address,
        Port:Endpoint.Port,
        Engine:Engine,
        EngineVersion:EngineVersion
    }' \
    --output table

echo ""
if aws rds describe-db-instances --db-instance-identifier syncflow-db --region $AWS_REGION --query 'DBInstances[0].DBInstanceStatus' --output text 2>/dev/null | grep -q "modifying"; then
    echo "⏳ Modification in progress. This usually takes 5-10 minutes."
    echo ""
    echo "Run this script again to check status:"
    echo "  ./check-rds-status.sh"
elif aws rds describe-db-instances --db-instance-identifier syncflow-db --region $AWS_REGION --query 'DBInstances[0].PubliclyAccessible' --output text 2>/dev/null | grep -q "False"; then
    echo "✅ RDS is now private (not publicly accessible)"
    echo ""
    echo "Note: You can only access RDS from:"
    echo "  - ECS tasks in the same VPC (your backend will still work!)"
    echo "  - EC2 instances in the same VPC"
    echo "  - Other resources within the VPC"
    echo ""
    echo "To access from local machine, use a bastion host or temporarily make it public."
else
    echo "✅ RDS is publicly accessible"
    echo ""
    echo "To make it private:"
    echo "  aws rds modify-db-instance --db-instance-identifier syncflow-db --no-publicly-accessible --apply-immediately --region ap-south-1"
fi

