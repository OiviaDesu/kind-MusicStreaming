---
name: Bug Report
about: Report a bug in MusicService Operator
title: '[BUG] '
labels: bug
assignees: ''
---

## Bug Description
A clear and concise description of the bug.

## Environment
- **Kubernetes Version**: (e.g., v1.28.0)
- **Operator Version**: (e.g., v0.1.0)
- **OS**: (e.g., Ubuntu 22.04)
- **Go Version**: (e.g., 1.22.0)

## Steps to Reproduce
1. Apply this MusicService YAML:
```yaml
# Paste your MusicService YAML here
```

2. Run command:
```sh
kubectl get musicservice <name> -o yaml
```

3. Observe error...

## Expected Behavior
What you expected to happen.

## Actual Behavior
What actually happened.

## Logs
```
# Controller logs
kubectl logs -n default deployment/musicservice-controller-manager

# Resource status
kubectl describe musicservice <name>
```

## Additional Context
Any other information that might help diagnose the issue.
