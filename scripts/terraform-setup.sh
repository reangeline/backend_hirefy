#!/bin/bash
set -e

echo "🚀 Setting up Terraform for ApplyWise..."

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

if ! command -v terraform &> /dev/null; then
    echo -e "${RED}❌ Terraform not found. Please install it first.${NC}"
    exit 1
fi

if ! command -v aws &> /dev/null; then
    echo -e "${RED}❌ AWS CLI not found. Please install it first.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Prerequisites OK${NC}"

# Create S3 bucket for state
echo -e "${YELLOW}Creating S3 bucket for Terraform state...${NC}"

BUCKET_NAME="applywise-terraform-state"
REGION="us-east-1"

# Check if bucket exists
if aws s3 ls "s3://${BUCKET_NAME}" 2>&1 | grep -q 'NoSuchBucket'; then
    echo "Creating bucket..."
    aws s3 mb "s3://${BUCKET_NAME}" --region ${REGION}
    
    # Enable versioning
    aws s3api put-bucket-versioning \
        --bucket ${BUCKET_NAME} \
        --versioning-configuration Status=Enabled
    
    # Enable encryption
    aws s3api put-bucket-encryption \
        --bucket ${BUCKET_NAME} \
        --server-side-encryption-configuration '{
            "Rules": [{
                "ApplyServerSideEncryptionByDefault": {
                    "SSEAlgorithm": "AES256"
                }
            }]
        }'
    
    # Block public access
    aws s3api put-public-access-block \
        --bucket ${BUCKET_NAME} \
        --public-access-block-configuration \
        "BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true"
    
    echo -e "${GREEN}✅ S3 bucket created and configured${NC}"
else
    echo -e "${GREEN}✅ S3 bucket already exists${NC}"
fi

# Optional: Create DynamoDB table for state locking
echo -e "${YELLOW}Creating DynamoDB table for state locking...${NC}"

TABLE_NAME="terraform-state-lock"

if aws dynamodb describe-table --table-name ${TABLE_NAME} 2>&1 | grep -q 'ResourceNotFoundException'; then
    aws dynamodb create-table \
        --table-name ${TABLE_NAME} \
        --attribute-definitions AttributeName=LockID,AttributeType=S \
        --key-schema AttributeName=LockID,KeyType=HASH \
        --billing-mode PAY_PER_REQUEST \
        --region ${REGION}
    
    echo -e "${GREEN}✅ DynamoDB table created${NC}"
else
    echo -e "${GREEN}✅ DynamoDB table already exists${NC}"
fi

# Create secrets.tfvars if not exists
cd terraform/environments/dev

if [ ! -f secrets.tfvars ]; then
    echo -e "${YELLOW}Creating secrets.tfvars from example...${NC}"
    cp secrets.tfvars.example secrets.tfvars
    echo -e "${RED}⚠️  Please edit secrets.tfvars with your actual credentials!${NC}"
fi

# Initialize Terraform
echo -e "${YELLOW}Initializing Terraform...${NC}"
terraform init

echo ""
echo -e "${GREEN}✅ Setup complete!${NC}"
echo ""
echo "Next steps:"
echo "1. Edit terraform/environments/dev/secrets.tfvars with your credentials"
echo "2. Build your Lambda: cd ../../../ && make build"
echo "3. Run: terraform plan -var-file=\"secrets.tfvars\""
echo "4. Run: terraform apply -var-file=\"secrets.tfvars\""

# scripts/get-stripe-keys.sh
#!/bin/bash
echo "🔑 Getting Stripe API Keys..."
echo ""
echo "1. Go to: https://dashboard.stripe.com/apikeys"
echo "2. Copy your 'Secret key' (starts with sk_test_...)"
echo "3. Go to: https://dashboard.stripe.com/webhooks"
echo "4. Create an endpoint or view existing"
echo "5. Copy the 'Signing secret' (starts with whsec_...)"
echo ""
echo "Add these to your secrets.tfvars:"
echo "stripe_secret_key     = \"sk_test_...\""
echo "stripe_webhook_secret = \"whsec_...\""
