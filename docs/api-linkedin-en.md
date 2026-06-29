# LinkedIn API Setup Guide for postctl

This document describes step-by-step how to create a LinkedIn Developer App, migrate it to the modern OpenID Connect (OIDC), and set up your credentials for `postctl`.

---

## 1. Create a Developer App on LinkedIn

1. Go to the [LinkedIn Developer Portal](https://linkedin.com/developers).
2. Log in with your personal LinkedIn account.
3. Click on **Create App**:
   * **App Name**: Give your app a name (e.g., `postctl-publisher`).
   * **LinkedIn Page**: Associate the app with your LinkedIn Company Page or create a temporary one (mandatory field).
   * **App Logo**: Upload the generated logo from `postctl/icons` (or any square image).
   * **Legal Terms**: Accept the terms and click **Create app**.

---

## 2. Add Products (Very Important!)

By default, a new LinkedIn app does not have permissions for posting updates. Since August 2023, LinkedIn replaced the legacy permissions (`r_liteprofile`). You must activate the following products:

1. In your app, go to the **Products** tab.
2. Search for **Share on LinkedIn** and click **Request access** (usually approved instantly).
3. Search for **Sign In with LinkedIn using OpenID Connect** (do not choose the outdated *Sign In with LinkedIn*) and activate it to unlock the modern OIDC login flow.

---

## 3. Configure Authentication Settings

1. Go to the **Auth** tab.
2. Scroll to the **OAuth 2.0 settings** section:
   * Under **Authorized Redirect URLs**, add the following callback:
     `http://localhost:8753/callback`
   * Click **Update**.
3. In the **OAuth 2.0 scopes** section, verify that you see the following scopes:
   * `openid`, `profile`, `w_member_social`, `email`.

---

## 4. Copy Client ID & Client Secret

1. Stay in the **Auth** tab.
2. In the **Application credentials** section, copy the **Client ID** and the **Client Secret**.

---

## 5. Configure in `postctl`

Use the CLI to save the credentials in your `config.yaml` (use the local binary `./postctl`):

```bash
# Set Client ID
./postctl config set linkedin.client_id "YOUR_CLIENT_ID"

# Set Client Secret
./postctl config set linkedin.client_secret "YOUR_CLIENT_SECRET"
```

Verify the settings:
```bash
./postctl config show
```

---

## 6. Establish Connection (Auth Flow)

Start the authentication flow:

```bash
./postctl auth linkedin
```

1. A browser window will automatically open, asking you to authorize your app to access your LinkedIn profile.
2. After clicking **Allow**, you will be redirected to the local web server.
3. The CLI intercepts the token via the OIDC flow, determines your user ID (via `/v2/userinfo`), encrypts the access token, and saves it in the local SQLite database.
4. You are now ready to publish posts on LinkedIn!
