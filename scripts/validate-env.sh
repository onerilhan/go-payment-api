#!/bin/bash

# Environment Configuration Validator
# Usage: ./scripts/validate-env.sh [development|staging|production]

ENV_TYPE=${1:-development}
ENV_FILE=".env.${ENV_TYPE}"

echo " Validating environment configuration: ${ENV_TYPE}"
echo " Environment file: ${ENV_FILE}"

# Check if env file exists
if [ ! -f "${ENV_FILE}" ]; then
    echo "❌ Error: Environment file ${ENV_FILE} not found!"
    exit 1
fi

# Load environment variables
set -a
source "${ENV_FILE}"
set +a

# Validation functions
validate_required_var() {
    local var_name=$1
    local var_value=${!var_name}
    
    if [ -z "${var_value}" ]; then
        echo "❌ Error: Required variable ${var_name} is not set or empty"
        return 1
    else
        echo " ${var_name} is set"
        return 0
    fi
}

validate_password_strength() {
    local var_name=$1
    local var_value=${!var_name}
    local min_length=${2:-16}
    
    if [ ${#var_value} -lt ${min_length} ]; then
        echo "  Warning: ${var_name} should be at least ${min_length} characters"
        return 1
    fi
    
    if [[ "${var_value}" =~ ^(password|123|admin|test|change|sample).*$ ]]; then
        echo " Critical: ${var_name} appears to be a default/weak password!"
        return 1
    fi
    
    echo " ${var_name} strength OK"
    return 0
}

validate_port() {
    local var_name=$1
    local var_value=${!var_name}
    
    if ! [[ "${var_value}" =~ ^[0-9]+$ ]] || [ "${var_value}" -lt 1 ] || [ "${var_value}" -gt 65535 ]; then
        echo " Error: ${var_name}=${var_value} is not a valid port number"
        return 1
    else
        echo " ${var_name} port OK"
        return 0
    fi
}

# Start validation
echo ""
echo " Validating required variables..."

ERRORS=0

# Required variables for all environments
validate_required_var "APP_ENV" || ERRORS=$((ERRORS+1))
validate_required_var "PORT" || ERRORS=$((ERRORS+1))
validate_required_var "DB_HOST" || ERRORS=$((ERRORS+1))
validate_required_var "DB_USER" || ERRORS=$((ERRORS+1))
validate_required_var "DB_PASSWORD" || ERRORS=$((ERRORS+1))
validate_required_var "DB_NAME" || ERRORS=$((ERRORS+1))
validate_required_var "JWT_SECRET" || ERRORS=$((ERRORS+1))

echo ""
echo " Validating port numbers..."

validate_port "PORT" || ERRORS=$((ERRORS+1))
validate_port "DB_PORT" || ERRORS=$((ERRORS+1))

echo ""
echo " Validating password security..."

# Different security requirements per environment
case "${ENV_TYPE}" in
    "production")
        echo " Production environment - strict security validation"
        validate_password_strength "DB_PASSWORD" 32 || ERRORS=$((ERRORS+1))
        validate_password_strength "JWT_SECRET" 64 || ERRORS=$((ERRORS+1))
        
        # Check for production-specific warnings
        if [[ "${DB_PASSWORD}" == *"CHANGE_THIS"* ]]; then
            echo " CRITICAL: Production DB password contains template text!"
            ERRORS=$((ERRORS+1))
        fi
        
        if [[ "${JWT_SECRET}" == *"CHANGE_THIS"* ]]; then
            echo " CRITICAL: Production JWT secret contains template text!"
            ERRORS=$((ERRORS+1))
        fi
        ;;
        
    "staging")
        echo " Staging environment - moderate security validation"
        validate_password_strength "DB_PASSWORD" 16 || ERRORS=$((ERRORS+1))
        validate_password_strength "JWT_SECRET" 32 || ERRORS=$((ERRORS+1))
        ;;
        
    "development")
        echo "  Development environment - basic validation"
        validate_password_strength "DB_PASSWORD" 8 || ERRORS=$((ERRORS+1))
        validate_password_strength "JWT_SECRET" 16 || ERRORS=$((ERRORS+1))
        ;;
esac

echo ""
echo " Validation Summary:"

if [ ${ERRORS} -eq 0 ]; then
    echo " All validations passed! Environment configuration is ready."
    echo " You can safely start the application with:"
    echo "   docker-compose -f docker-compose.yml --env-file ${ENV_FILE} up -d"
    exit 0
else
    echo " Found ${ERRORS} validation error(s). Please fix them before proceeding."
    echo ""
    echo " Common fixes:"
    echo "   - Generate strong passwords: openssl rand -base64 32"
    echo "   - Generate JWT secret: openssl rand -base64 64"
    echo "   - Update template values in ${ENV_FILE}"
    exit 1
fi