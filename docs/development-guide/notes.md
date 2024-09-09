---
title: Development notes
summary: Some notes for developers of the code.
authors:
  - hikarukin
date: 2024-07-04
---

Developer notes
===============

- Because of dependency issues we decided to use a stub instead of importing all
  dependencies behind the original code of ArgoCD.
  
  More info in `internal/stubs/argocd/v1beta1`
