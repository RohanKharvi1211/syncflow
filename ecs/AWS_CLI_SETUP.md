# AWS CLI Setup Guide

## Installation

AWS CLI is now installed! Verify it works:

```bash
aws --version
```

## Configuration

Before you can use AWS CLI commands, you need to configure it with your AWS credentials.

### Option 1: Quick Setup (Interactive)

Run this command and follow the prompts:

```bash
aws configure
```

You'll be prompted for:
1. **AWS Access Key ID**: Your AWS access key
2. **AWS Secret Access Key**: Your AWS secret key
3. **Default region name**: e.g., `us-east-1`, `us-west-2`, `eu-west-1`
4. **Default output format**: `json` (recommended) or `text` or `table`

### Option 2: Manual Setup

If you prefer to set credentials manually:

```bash
# Create AWS credentials directory
mkdir -p ~/.aws

# Create credentials file
cat > ~/.aws/credentials << EOF
[default]
aws_access_key_id = YOUR_ACCESS_KEY_ID
aws_secret_access_key = YOUR_SECRET_ACCESS_KEY
EOF

# Create config file
cat > ~/.aws/config << EOF
[default]
region = us-east-1
output = json
EOF
```

### Option 3: Environment Variables

You can also set credentials as environment variables:

```bash
export AWS_ACCESS_KEY_ID=YOUR_ACCESS_KEY_ID
export AWS_SECRET_ACCESS_KEY=YOUR_SECRET_ACCESS_KEY
export AWS_DEFAULT_REGION=us-east-1
```

**Note:** For this session only. Add to `~/.zshrc` to make permanent.

## Getting Your AWS Credentials

### If you don't have AWS credentials yet:

1. **Create an AWS Account** (if you don't have one):
   - Go to https://aws.amazon.com/
   - Sign up for an account

2. **Create an IAM User** (recommended, not your root account):
   - Go to AWS Console > IAM > Users
   - Click "Create user"
   - Give it a name (e.g., `syncflow-admin`)
   - Select "Provide user access to the AWS Management Console" (optional) or "Programmatic access"
   - Attach policies: `AdministratorAccess` (for initial setup) or more restrictive policies
   - Create user and **save the access key ID and secret access key**

3. **For CI/CD purposes**, create a separate IAM user:
   ```bash
   aws iam create-user --user-name syncflow-cicd
   aws iam create-access-key --user-name syncflow-cicd
   aws iam attach-user-policy --user-name syncflow-cicd --policy-arn arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryPowerUser
   aws iam attach-user-policy --user-name syncflow-cicd --policy-arn arn:aws:iam::aws:policy/AmazonECS_FullAccess
   ```

## Verify Your Setup

Test your AWS CLI configuration:

```bash
# Check your AWS identity
aws sts get-caller-identity

# List your IAM users
aws iam list-users

# Check your default region
aws configure get region
```

Expected output for `get-caller-identity`:
```json
{
    "UserId": "AIDAXXXXXXXXXXXXXXXXX",
    "Account": "123456789012",
    "Arn": "arn:aws:iam::123456789012:user/your-username"
}
```

## Next Steps

Once AWS CLI is configured, you can:

1. **Create AWS Secrets Manager secrets**:
   ```bash
   cd ecs
   ./create-secrets.sh
   ```

2. **Setup infrastructure**:
   ```bash
   cd ecs
   ./setup-infrastructure.sh
   ```

3. **Get secret ARNs**:
   ```bash
   cd ecs
   ./get-secret-arns.sh
   ```

## Troubleshooting

### Error: "Unable to locate credentials"

**Solution:** Run `aws configure` to set up your credentials.

### Error: "Access Denied"

**Solution:** Your IAM user doesn't have the necessary permissions. Attach appropriate policies:
- `AmazonEC2ContainerRegistryPowerUser` - For ECR access
- `AmazonECS_FullAccess` - For ECS access
- `IAMFullAccess` - For creating IAM roles (or create custom policy)
- `SecretsManagerReadWrite` - For Secrets Manager access

### Error: "Region not specified"

**Solution:** Set default region:
```bash
aws configure set region us-east-1
```

Or specify region in commands:
```bash
aws --region us-east-1 <command>
```

### Multiple AWS Profiles

If you work with multiple AWS accounts, you can use profiles:

```bash
# Configure a profile
aws configure --profile syncflow-prod

# Use a profile
aws s3 ls --profile syncflow-prod

# Set default profile
export AWS_PROFILE=syncflow-prod
```

## Security Best Practices

1. **Never commit AWS credentials to Git** - They're stored in `~/.aws/credentials`
2. **Use IAM users with minimal permissions** - Don't use root account credentials
3. **Rotate credentials regularly** - Change access keys periodically
4. **Enable MFA** - For additional security
5. **Use roles for ECS tasks** - Not access keys

## Additional Resources

- [AWS CLI User Guide](https://docs.aws.amazon.com/cli/latest/userguide/)
- [AWS CLI Command Reference](https://docs.aws.amazon.com/cli/latest/reference/)
- [IAM Best Practices](https://docs.aws.amazon.com/IAM/latest/UserGuide/best-practices.html)

