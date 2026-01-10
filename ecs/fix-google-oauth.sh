#!/bin/bash
# Script to create task definition with Google OAuth secrets and redirect URI

set -e

AWS_REGION=${AWS_REGION:-ap-south-1}
PROJECT_NAME=${PROJECT_NAME:-syncflow}

echo "Creating task definition with Google OAuth secrets..."

# Get ALB DNS
ALB_DNS=$(aws elbv2 describe-load-balancers --names "${PROJECT_NAME}-backend-alb" --region $AWS_REGION --query 'LoadBalancers[0].DNSName' --output text)
REDIRECT_URI="http://${ALB_DNS}/api/oauth/google/callback"

# Get task definition revision 15 (has Google secrets)
aws ecs describe-task-definition --task-definition "${PROJECT_NAME}-backend-task:15" --region $AWS_REGION --query 'taskDefinition' > /tmp/base-td.json

# Add GOOGLE_REDIRECT_URI using Python
python3 << PYTHON
import json

with open('/tmp/base-td.json') as f:
    td = json.load(f)

container = td['containerDefinitions'][0]

if 'environment' not in container:
    container['environment'] = []

# Add or update GOOGLE_REDIRECT_URI
redirect_uri = '${REDIRECT_URI}'
found = False
for env in container['environment']:
    if env['name'] == 'GOOGLE_REDIRECT_URI':
        env['value'] = redirect_uri
        found = True
        break

if not found:
    container['environment'].append({
        'name': 'GOOGLE_REDIRECT_URI',
        'value': redirect_uri
    })

# Remove read-only fields
for key in ['taskDefinitionArn', 'revision', 'status', 'requiresAttributes', 
            'placementConstraints', 'compatibilities', 'registeredAt', 'registeredBy']:
    td.pop(key, None)

with open('/tmp/final-td.json', 'w') as f:
    json.dump(td, f, indent=2)

print(f"✓ Created task definition with GOOGLE_REDIRECT_URI={redirect_uri}")
PYTHON

# Register new task definition
NEW_REV=$(aws ecs register-task-definition --cli-input-json file:///tmp/final-td.json --region $AWS_REGION --query 'taskDefinition.revision' --output text)
echo "✓ Registered as revision: $NEW_REV"

# Update service with force deployment
echo "Updating service to use new task definition..."
aws ecs update-service \
    --cluster "${PROJECT_NAME}-cluster" \
    --service "${PROJECT_NAME}-backend-service" \
    --task-definition "${PROJECT_NAME}-backend-task:${NEW_REV}" \
    --force-new-deployment \
    --region $AWS_REGION \
    --query 'service.{Status:status,TaskDefinition:taskDefinition}' \
    --output table

echo ""
echo "✓ Service updated! Wait 1-2 minutes for new task to start."
echo "Redirect URI: $REDIRECT_URI"
echo ""
echo "⚠️  Don't forget to add this redirect URI to Google Cloud Console OAuth credentials!"

