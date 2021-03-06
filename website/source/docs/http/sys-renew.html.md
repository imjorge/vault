---
layout: "http"
page_title: "HTTP API: /sys/renew"
sidebar_current: "docs-http-lease-renew"
description: |-
  The `/sys/renew` endpoint is used to renew secrets.
---

# /sys/renew

<dl>
  <dt>Description</dt>
  <dd>
    Renew a secret, requesting to extend the lease.
  </dd>

  <dt>Method</dt>
  <dd>PUT</dd>

  <dt>URL</dt>
  <dd>`/sys/renew/<lease id>`</dd>

  <dt>Parameters</dt>
  <dd>
    <ul>
      <li>
        <span class="param">increment</span>
        <span class="param-flags">optional</span>
        A requested amount of time in seconds to extend the lease.
        This is advisory.
      </li>
    </ul>
  </dd>

  <dt>Returns</dt>
  <dd>A secret structure.
  </dd>
</dl>
