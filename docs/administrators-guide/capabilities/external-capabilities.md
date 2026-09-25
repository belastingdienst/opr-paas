---
title: External capabilities
summary: A detailed description of what external capabilities are and what you could do with them
authors:
  - Devotional Phoenix
date: 2026-09-21
---

# External capabilities 

In release v3.13.0 External capabilities were added. This enables the creation of Capabilities without quota or a namespace.
The main goal for external capabilities is to add a capability which is fully managed outside the paas operator (systems only relying on the argocd plugin-generator integration)
One use-case could be a tool which creates Harbor projects by scanning the ArgoCD plugin Generator output for Paas'es with the harbor capability enabled.
By defining the harbor capability as an external capability, the Paas'es with the harbor capability enabled end up in the ArgoCD plugin generator output, but no k8s resources are created.
A capability becomes external when all 3 quota settings are set to nil, or no quota block is defined.
