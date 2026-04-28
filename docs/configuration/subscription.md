### Structure

```json
{
  "name": "",
  "url": "",
  "user_agent": "",
  "request_header": {
    "Authorization": "",
    "X-Subscription": ""
  },
  "decrypt": {
    "key": "",
    "response_header": "",
    "response_header_value": ""
  },
  "process": [
    {
      "filter": [],
      "exclude": [],
      "filter_type": [],
      "exclude_type": [],
      "invert": false,
      "remove": false,
      "rename": {},
      "remove_emoji": false,
      "rewrite_multiplex": {}
    }
  ],
  "deduplication": false,
  "update_interval": "5m",
  "generate_selector": false,
  "generate_urltest": false,
  "urltest_suffix": "",
  "custom_selector": {},
  "custom_urltest": {}
}
```

### Fields

#### name

==Required==

Name of the subscription, will be used in group tags.

#### url

==Required==

Subscription URL.

#### user_agent

User-Agent in HTTP request.

`serenity/$version (sing-box $sing-box-version; Clash compatible)` is used by default.

#### request_header

Custom HTTP request headers sent when fetching the subscription.

This is useful for authenticated subscription endpoints, for example:

```json
{
  "request_header": {
    "Authorization": "Bearer <token>",
    "X-Subscription": "example"
  }
}
```

If `user_agent` is also configured, it overrides `request_header.User-Agent`.

#### decrypt

Subscription response body decryption options.

If the response header matches `decrypt.response_header: decrypt.response_header_value`,
Serenity will decrypt the response body before parsing the subscription.

Currently the supported format is:

* `Base64(IV + AES-256-CBC(ciphertext))`
* IV = first 16 bytes after Base64 decoding
* PKCS7 padding

If the response is marked as encrypted but `decrypt` is not configured,
subscription update will fail with an explicit error.

#### decrypt.key

32-byte AES key.

#### decrypt.response_header

Response header name that indicates the body is encrypted.

`x-flclash-encrypted` is used by default.

#### decrypt.response_header_value

Expected response header value that enables decryption.

`1` is used by default.

#### process

!!! note ""

    You can ignore the JSON Array [] tag when the content is only one item

Process rules.

#### process.filter

Regexp filter rules, match outbound tag name.

#### process.exclude

Regexp exclude rules, match outbound tag name.

#### process.filter_type

Filter rules, match outbound type.

#### process.exclude_type

Exclude rules, match outbound type.

#### process.invert

Invert filter results.

#### process.remove

Remove outbounds that match the rules.

#### process.rename

Regexp rename rules, matching outbounds will be renamed.

#### process.remove_emoji

Remove emojis in outbound tags.

#### process.rewrite_multiplex

Rewrite [Multiplex](https://sing-box.sagernet.org/configuration/shared/multiplex) options.

#### deduplication

Remove outbounds with duplicate server destinations (Domain will be resolved to compare).

#### update_interval

Subscription update interval.

`1h` is used by default.

#### generate_selector

Generate a global `Selector` outbound for the subscription.

If both `generate_selector` and `generate_urltest` are disabled, subscription outbounds will be added to global groups.

#### generate_urltest

Generate a global `URLTest` outbound for the subscription.

If both `generate_selector` and `generate_urltest` are disabled, subscription outbounds will be added to global groups.

#### urltest_suffix

Tag suffix of generated `URLTest` outbound.

` - URLTest` is used by default.

#### custom_selector

Custom [Selector](https://sing-box.sagernet.org/configuration/outbound/selector/) template.

#### custom_urltest

Custom [URLTest](https://sing-box.sagernet.org/configuration/outbound/urltest/) template.
