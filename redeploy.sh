#!/bin/bash

# Activate virtual environment using pipenv
echo "Activating virtual environment..."
eval "$(pipenv --venv)/bin/activate"

# Stop localstack
echo "Stopping localstack..."
localstack stop

# Remove deployment directories
echo "Cleaning up deployment directories..."
rm -rf .chalice/deployments
rm -rf .chalice/deployed

# Start localstack
echo "Starting localstack..."
localstack start -d

# Wait for localstack to be ready
echo "Waiting for localstack to be ready..."
sleep 10

# Create resources
echo "Creating resources..."
python scripts/create_resources.py

# Deploy with chalice
echo "Deploying with chalice..."
chalice deploy --stage dev

echo "Redeployment complete!"
