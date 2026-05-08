#!/bin/bash
# This script runs inside LocalStack after startup via the init hook.
# Terraform handles real provisioning — this is just a readiness signal.
echo "==> LocalStack init hook executed. Terraform will provision resources."
