# Authentication and Routing Keys

## Motivation

The baseline router trusts `X-Routing-Key` unconditionally. This is an explicit non-goal: authentication is assumed to happen upstream, and the router's job is purely routing logic. But production systems must answer: **where is the trust boundary, and how is routing metadata validated?**

This annex explores options for securing routing keys, where authentication fits in the request path, and why separating auth from routing logic matters.