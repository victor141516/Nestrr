# webOS playback smoke test

Small hosted app for **login → downloaded movie → playback → Back**. The launcher
loads the same Nestrr website over HTTPS; credentials remain in the existing login
flow and are never embedded in the package. `?tv=1` is outside Nestrr's hash router
and opts into the remote behavior. Nestrr passes it into Plarr's player URL.

## Build

From the Nestrr repository in PowerShell:

```powershell
.\apps\webos\package.ps1
```

This installs LG's pinned `@webos-tools/cli@3.2.6` into the ignored
`.runtime/webos-tools` folder and creates
`.runtime/webos/site.viti.nestrr.smoke_0.0.1_all.ipk`. The default origin is
`https://nestrr.viti.site`. Use `-ServerOrigin 'http://your-development-host:8080'`
to target another instance; its Plarr public origin must also be reachable by
the TV. Deploy the corresponding Nestrr **and** Plarr frontend changes before
installing the package. The launcher alone does not add remote behavior to an
older server deployment.

## Install on a TV

First enable LG's Developer Mode app and Key Server on the TV. Obtain the actual
model/webOS version and IP address. Keep the PC and TV on a reachable network.

```powershell
$cli = '.\.runtime\webos-tools\node_modules\.bin'
& "$cli\ares-setup-device.cmd" --add nestrr-tv -i 'host=TV_IP' -i 'port=9922' -i 'username=prisoner'
& "$cli\ares-novacom.cmd" --device nestrr-tv --getkey
# Enter the passphrase displayed on the TV when prompted; do not save it here.
& "$cli\ares-device.cmd" --device nestrr-tv --system-info
& "$cli\ares-install.cmd" --device nestrr-tv '.\.runtime\webos\site.viti.nestrr.smoke_0.0.1_all.ipk'
& "$cli\ares-launch.cmd" --device nestrr-tv site.viti.nestrr.smoke
```

## Check on the real TV

1. Sign in with the existing account using the TV keyboard. Do not record the
   password in screenshots or diagnostic output.
2. Select an already downloaded movie with arrows and OK. The selected control
   has an accent outline. The detail view initially focuses Play/Resume.
3. Press OK to open Plarr. Confirm picture, sound and advancing playback. OK on
   the video pauses/resumes; existing left/right shortcuts seek by five seconds.
4. Press Back. Any player menu closes first; otherwise playback closes and focus
   returns to Play/Resume. A second Back returns to the previous movie list and
   restores the selected card. At the app entry page Back exits the test app.
5. Record the TV model, firmware/webOS version, selected playback engine,
   observed audio/video result, and any browser console errors. Repeat after
   relaunch to check login persistence. Keep deliberate test clips under ten
   seconds; stop movie playback promptly after observing it.

## Scope and verification

The first slice adds remote focus/navigation to Nestrr and OK/Back/media-key
handling to Plarr. It uses Plarr's existing engine selection and transcoding.
Full directional navigation of Plarr's quality/audio/subtitle menus, TV memory
tuning, older web engines, native-HLS parity, HDR certification, and standby
recovery are subsequent work guided by the actual device result.

`scripts/e2e.ps1 -KeepRuntime` includes `e2e/runner/television-smoke.mjs`: it uses
a fresh browser context to log in using keyboard input, open an eight-second
fixture, verify video advances, pause/resume with OK, dispatch LG Back (461),
and check focus restoration to the detail and list. This verifies browser
behavior, **not** the LG decoder, on-screen keyboard, hardware HDR, or installation.

The package is a Developer Mode test app, not an LG store release. Developer
Mode sessions expire and installed development apps can be removed when the mode
is disabled.

References: [hosted apps](https://webostv.developer.lge.com/develop/getting-started/web-app-types),
[Back](https://webostv.developer.lge.com/develop/guides/back-button),
[Developer Mode](https://webostv.developer.lge.com/develop/getting-started/developer-mode-app),
[engine versions](https://webostv.developer.lge.com/develop/specifications/web-api-and-web-engine).
