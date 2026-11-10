# App access: reaching Rezeptplaner from your phone

**English** · [Deutsch](APP-ZUGANG.md)

The Android app doesn't talk to Rezeptplaner through Home Assistant but through its own **app access** on **port 8100**. There the server requires a valid API key with **every** request (`Authorization: Bearer rp_…`) and answers 401 otherwise. The web interface and the key management aren't available on that port.

The web interface (port 8099, Ingress) stays separate and is only reachable through Home Assistant.

In the examples, `recipes.example.org` stands for your host name.

> **Never forward port 8100 – or any other Home Assistant port – directly to the internet on your router.** Use one of the two ways below: a VPN into your home network, or a tunnel.

## 1. Create an API key

Open the web interface → **Settings → Install the app**:

1. **Step 1 – Install the app:** QR codes for Obtainium and for Rezeptplaner's entry in Obtainium (or the APK directly).
2. **Step 2 – Connect the app:** Enter the **Address for the app** once and save it – the address your phone uses to reach port 8100 (see sections 2 and 3), e.g. `https://recipes.example.org`. Then **Connect new device** → enter a device name (e.g. “Alex's phone”) → **Create QR code**. Only now is a key created.
3. In the app, choose to set it up and scan the QR code. If Cloudflare Access is active (section 5), first enter Client ID and Secret under **Cloudflare Access (optional)**; they only go into the QR code and aren't stored.
4. QR code and key are shown only this once (until **Done** or until you leave the page); only a hash of the key is stored.

Lock out a lost or retired device with **Revoke** in the same place. The key stops working immediately.

## 2. Option A: VPN

With a VPN (e.g. WireGuard, Tailscale, or your router's VPN), your phone is part of your home network while away. Nothing is exposed to the internet.

1. Under **Settings → Apps → Rezeptplaner → Configuration → Network**, map port **8100** to a host port (e.g. 8100). It's then reachable inside your home network at e.g. `http://192.168.1.20:8100`.
2. **The app only accepts `https://` addresses.** Put an HTTPS endpoint in front of port 8100 inside your network – for example a reverse proxy app with a certificate for a local host name, or Tailscale Serve – and use its address, e.g. `https://recipes.home.example.org`, as the **Address for the app**.

If that's too much effort, the tunnel in option B is the simpler way.

## 3. Option B: Cloudflare tunnel

Port 8100 doesn't need to be opened at all for this: the Cloudflared app runs in the same internal Supervisor network and reaches Rezeptplaner by its host name.

1. Look up the app's host name: **Settings → Apps → Rezeptplaner**; the app information shows the **Hostname**, e.g. `45a92d34-rezeptplaner`.
2. Add another host name to the configuration of the Cloudflared app (depending on the app, under `additional_hosts`):

   ```yaml
   additional_hosts:
     - hostname: recipes.example.org
       service: http://45a92d34-rezeptplaner:8100
   ```

   For a tunnel managed in the Cloudflare dashboard: **Zero Trust → Networks → Tunnels → your tunnel → Public Hostname → Add**, service `HTTP`, URL `45a92d34-rezeptplaner:8100`.
3. Restart the Cloudflared app.

Check it (Linux, macOS or Git Bash):

```bash
# without a key: 401
curl -s -o /dev/null -w "%{http_code}\n" https://recipes.example.org/api/v1/health
# with a key: 200
curl -s -H "Authorization: Bearer rp_YOUR_KEY" https://recipes.example.org/api/v1/health
```

## 4. Cache rule

As with the Home Assistant host name, Cloudflare shouldn't cache anything here: **Caching → Cache Rules**, condition *Hostname equals* `recipes.example.org`, action **Bypass cache**. (The server sends `Cache-Control: private` for images anyway.)

## 5. Extra protection with Cloudflare Access (recommended for tunnels)

With Cloudflare Access, Cloudflare only lets requests with a valid **service token** through to the tunnel. Without a token, Cloudflare itself answers 403 and Rezeptplaner never sees the request. A stolen API key alone is then no longer enough.

1. **Zero Trust → Access → Service Auth → Service Tokens → Create Service Token**, name e.g. “Rezeptplaner app”, duration as you like. Note down **Client ID** and **Client Secret** (the secret is shown only once).
2. **Zero Trust → Access → Applications → Add an application → Self-hosted**:
   - name “Rezeptplaner app”, domain `recipes.example.org`
   - policy: action **Service Auth**, Include → **Service Token** → the token from step 1
3. Save.

Check it:

```bash
# without a service token: 403 from Cloudflare
curl -s -o /dev/null -w "%{http_code}\n" https://recipes.example.org/api/v1/health
# with service token and API key: 200
curl -s https://recipes.example.org/api/v1/health \
  -H "CF-Access-Client-Id: YOUR_CLIENT_ID.access" \
  -H "CF-Access-Client-Secret: YOUR_CLIENT_SECRET" \
  -H "Authorization: Bearer rp_YOUR_KEY"
```

The Android app can send **CF-Access-Client-Id** and **CF-Access-Client-Secret** (enter them when connecting, or later under **Settings → Cloudflare Access** in the app). Without them it only works as long as Access isn't active for the host name.
