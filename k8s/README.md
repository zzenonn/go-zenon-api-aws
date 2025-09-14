# Kubernetes Manifests for go-zenon-api-aws

This directory contains the Kubernetes manifests required to deploy the go-zenon-api-aws application on Kubernetes. All resources are defined in a single manifest for simplicity.

## Contents

- **go-service.yaml**: Contains all required Kubernetes resources:
  - Namespace
  - ConfigMap for IRSA configuration
  - ServiceAccount for IRSA (IAM Roles for Service Accounts)
  - Deployment for the application
  - Service to expose the application

## Usage

1. Update `go-service.yaml` as needed for your environment (image, namespace, AWS IAM role ARN, etc).
2. Deploy all resources with a single command:

```sh
kubectl apply -f go-service.yaml
```

## IRSA (IAM Roles for Service Accounts)

The ServiceAccount and ConfigMap are set up for AWS IAM Roles for Service Accounts (IRSA) integration. Update the `eks.amazonaws.com/role-arn` annotation in the ServiceAccount section of `go-service.yaml` to match your AWS IAM role ARN.

---

Feel free to further customize the manifest to fit your deployment needs. 