# Mobile app behind Cloudflare Access

Publishing the gateway through a Cloudflare Tunnel and gating it with
Cloudflare Access gives the web admin a strong front door: nothing
reaches the tunnel until Cloudflare has verified who you are. This
page covers making the mobile app pass the same gate, so the phone
works from cellular without a VPN and without opening a hole in the
Access policy.

## How it works

A browser passes Access by signing in once and keeping the
`CF_Authorization` cookie Cloudflare sets on the app hostname. A
native app has no such flow, so opendray runs the identical sign-in
inside an embedded WebView, lifts the resulting cookie out of the
platform cookie store, and attaches it to every request, including the
WebSocket handshake the session terminal uses.

Nothing about opendray's own authentication changes. Access decides
whether the request reaches the tunnel; opendray still decides whether
the caller is signed in. The two expire independently, so an Access
session rolling over does not sign you out of opendray.

Both secrets live in the platform keystore (iOS Keychain, Android
EncryptedSharedPreferences). Neither is compiled into the app.

## What you are accepting by using this

The sign-in runs in an embedded WebView, not in the OS browser. That
is not the shape RFC 8252 recommends for native apps, and it is worth
being explicit about why, and about what it costs.

It costs the browser's own address bar. When you type identity-provider
credentials into the sign-in screen, the hostname shown above the page
is drawn by opendray, not by Safari or Chrome, so it is only as
trustworthy as the app rendering it. A system browser would give you
chrome the app cannot forge.

The reason it is a WebView anyway is that the cookie is the entire
mechanism. Cloudflare Access proves who you are by setting
`CF_Authorization` on the gateway hostname, and a cookie set inside
Safari or Chrome cannot be read by the app. `ASWebAuthenticationSession`
and Android Custom Tabs are no help here for the same reason: they
deliberately keep their cookie jar away from the app. Without a
readable cookie there is no way for a native client to satisfy Access
short of a shared Service Token or a client certificate, which trade
per-user identity away for a static credential.

If that trade is not one you want to make, the alternative is to leave
the app on the LAN or on a VPN. There is no version of this feature
that is both WebView-free and cookie-based.

## Before you start: pick an identity provider the WebView accepts

This is the one real constraint, and it is worth settling first.

| Access login method | Works in the app's WebView |
|---|---|
| One-time PIN (email code) | Yes. Recommended. |
| GitHub | Yes |
| Generic OIDC / SAML (self-hosted IdP) | Usually |
| Google | No. Google refuses OAuth in embedded WebViews (`disallowed_useragent`). |
| Microsoft Entra ID | Often refused, for the same reason |

If your Access policy uses Google as the identity provider, the app's
sign-in will fail with a Google error page. Add One-time PIN as an
additional login method on the Access application; the web admin keeps
using Google, and the phone uses the emailed code. Faking a desktop
browser user-agent to get around the check is not supported here, and
Google's terms disallow it.

## Cloudflare side

1. In Zero Trust, open the Access application that already protects
   the gateway hostname.
2. Under **Login methods**, make sure at least one method from the
   "Yes" rows above is enabled.
3. Leave the application covering the whole hostname. Do not add a
   bypass for `/api/v1/*`. A bypass would let the app skip the
   WebView, at the cost of exposing the login endpoint and the entire
   API surface to the open internet, which is the thing Access is
   there to prevent.
4. Check the **session duration**. That is how often the phone has to
   repeat the sign-in. 24 hours is a reasonable starting point.

WebSockets need no special Access configuration: Cloudflare validates
the cookie on the upgrade request like any other.

## Phone side

On first launch, enter the public gateway URL as usual. When Access
challenges the probe, the app says so and opens the Cloudflare
sign-in. Finish it and onboarding continues to the normal opendray
login.

Later challenges are handled the same way from wherever they happen:
when the Access session expires, the sign-in appears over whatever
screen you were on, and the screen underneath keeps its state.

**Settings → Security** shows the current Access session and its
expiry, re-runs the sign-in on demand, and clears this device's Access
cookie. Pointing the app at a different gateway clears the cookie
automatically, since it is scoped to the old hostname.

"Clear Access session" means what it says and no more: it forgets the
cookie on this device, in both places it is stored. It does not end
the session at Cloudflare, and it does not sign you out of your
identity provider. If your IdP still has you signed in, the next
sign-in may complete without asking you for anything. To actually end
the Access session, sign out at your identity provider.

The session terminal and the live log tail run over WebSockets, which
Access gates like any other request. A WebSocket rejected by Access
looks identical to an unreachable gateway from the client's side, so
when either one drops, the app quietly re-checks over HTTP and raises
the sign-in if that is what the problem turns out to be.

## App lock

The same Security screen has **Require unlock**, off by default.

It exists because of what publishing the gateway costs. Once the app
reaches the gateway from anywhere, an unlocked phone is a live admin
console: the bearer token and the Access cookie are both already on
the device, so whoever holds it is past both gates. Turning the lock
on requires biometrics or the device passcode before the app is shown,
and again after the app has been in the background for more than a
minute.

The one-minute threshold is deliberate. Access sign-in routinely sends
you to a mail app for a one-time code, and a shorter threshold would
make the two features fight each other.

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| Sign-in page shows a Google "disallowed_useragent" error | Access is using Google as the identity provider | Add One-time PIN to the application's login methods |
| Sign-in completes but the app still says Access is required | The Access application covers a different hostname than the Gateway URL | Make the Gateway URL exactly the hostname the Access application protects |
| Terminal or live logs drop, then the sign-in appears | The Access session expired mid-stream | Finish the sign-in; the stream reconnects |
| Terminal never connects and no sign-in appears, everything else works | Something in the path is dropping the WebSocket upgrade | Not an Access problem; see [operator-guide §Topology](operator-guide.md#topology) |
| Sign-in reappears every few hours | Access session duration is short | Raise it on the Access application |
| Repeated sign-in prompts after cancelling | Expected: dismissing suppresses the prompt for 30 seconds, then it returns while requests keep failing | Finish the sign-in, or switch back to a LAN Gateway URL |
| Sign-in completes instantly without asking anything | A live cookie or IdP session was still there | Expected. To force a full re-check, sign out at your identity provider |

## See also

- [mobile-app.md](mobile-app.md): building and installing the app
- [operator-guide.md](operator-guide.md): reverse proxy and tunnel topology
