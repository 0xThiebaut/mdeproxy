# MDE Proxy
MDE Proxy leverages an undocumented proxy at [security.microsoft.com] to access the undocumented
Microsoft Defender for Endpoint APIs (e.g., device timeline).

This tool is inspired by [Defender Harvester] which does not rely on the undocumented proxy; However
some organizations prevent direct access to the Security Center API (i.e., conditional access policies) which
hinders incident response. This tool acts as a workaround: If you can access the timeline in a browser,
this tool can extract the data from the timeline API.

As a rough estimate, the timeline API produces 1GB of data per appliance per month;
Extraction of the data takes around 20 minutes per device per month.

## Getting Started

### Installation
MDE Proxy is written in [Go] and can be installed as follows...
```bash
go install github.com/0xThiebaut/mdeproxy@latest
```

### Configuration
MDE Proxy relies on two headers sent through [security.microsoft.com]:
- `Cookie` which holds authentication data
- `X-XSRF-TOKEN` which holds a cross-site request forgery token

Extracting these header values can be done through the browser's developer tools when inspecting `POST` requests.

![Capture]

### Usage
With the two header values extracted, a device's timeline can be extracted as follows...
```bash
mdeproxy timeline --cookie COOKIE --xsrf XSRF --machine MID --from 2024-04-01T00:00:00Z --to 2024-07-01T00:00:00Z --output timeline.jsonl
```
- `COOKIE` being the cookie header extracted through the browser's developer tools.
- `XSRF` being the cross-site request forgery token extracted through the browser's developer tools.
- `MID` being the hexadecimal machine ID.

The `from` and `to` field represents the time-range of timeline data to recover.
This tool handles paging and is hence not subject to the usual 7 or 30 day limit.
By omitting `from` and `to`, the last 6 months of data are exported.

[Defender Harvester]: https://github.com/olafhartong/DefenderHarvester
[Go]: https://go.dev/doc/install
[security.microsoft.com]: https://security.microsoft.com

[Capture]: docs/images/Tokens.jpg