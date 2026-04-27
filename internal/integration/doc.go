// Package integration is the registry + reverse proxy for external apps
// that consume opendray's API.
//
// Responsibilities (per design §8.2):
//   - Register / list / deregister integrations; issue scoped API keys.
//   - Reverse-proxy /api/v1/integrations/{prefix}/* to the integration's base_url.
//   - Health-check registered integrations on a 30s cadence.
//   - Push events over WebSocket to subscribed integrations.
//
// Implementation lands in M3. Pure registry contract — third-party code
// never enters this repo.
package integration
