---
title: Go Template introduction
summary: A detailed description of how Go Template is used to replace hardcoded options with dynamic PaasConfig values.
authors:
  - Devotional Phoenix
date: 2025-06-18
---

# Introduction

## Template

The template feature allows administrators to dynamically generate values from information in Paas and/or PaasConfig.
This provides flexibility for organizations using the Paas operator to define business specific logic.

## Syntax

Template options support standard Go Template syntax, allowing all values from the Paas and PaasConfig to be referenced. See more examples below.
In addition to the default Go Template functions, we've added support for
[all Sprout](https://docs.atom.codes/sprout/registries/list-of-all-registries) Go Template functions,
and [backward](https://docs.atom.codes/sprout/registries/backward) (we want to use the `fail` function.

## Behavior of multivalued and single valued results

Depending on the result of the Go Template, one of three things can happen:

- if the result can be parsed as a list:
  - The key of the resulting item (label or custom field) is derived from the name of the template, suffixed with an underscore and an integer (number in the list of this item).
  - The value of the resulting item is the direct value of the item in the list
- if the result can be parsed as map:
  - The key of the resulting item (label or custom field) is derived from the name of the template, suffixed with an underscore and the key of the map item
  - The value of the resulting item is the direct value of the map item
- If the result is not parsable as list or map:
  - The key of the resulting item (label or custom field) is derived from the name of the template
  - The value of the resulting item is derived from the exact returned string

!!! note

    If you need to return a map or list as a single string value in a field, you have the following options:
    - convert the map to a string representation using toYaml or toJson, and add quoting to make sure it is parsed as one string
    - create a map with one key/value pair and set the resulting string as the value

## Developing Go Templates

For easier validation and debugging of templates, we recommend using [Repeat It](https://repeatit.io/), an online tool to test and validate your Go Templates.
