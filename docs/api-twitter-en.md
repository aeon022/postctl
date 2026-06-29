# Twitter/X API Setup Guide for postctl

This document describes step-by-step how to set up authentication for Twitter/X in `postctl`.

Two options are available:
* **Option A: Cookie-based Authentication (Free but Unofficial)** – Uses your existing browser session. Free, but prone to errors and carries a risk of account suspension.
* **Option B: Official API (Paid & Recommended)** – Uses the official Twitter API (requires a paid subscription starting at ~$100/month or prepaid API credits). The secure, stable way.

---

## Option A: Cookie-based Authentication (Free but Unofficial)

This method simulates a real browser session by using your login cookies. It is completely free and requires no setup in the Twitter Developer Portal.

> [!WARNING]
> **Unofficial Bypass Method (Risk of Account Suspension):**
> Cookie-based authentication simulates a web session. This method is error-prone, violates X/Twitter's Terms of Service (ToS), and can lead to the suspension of your account. The only official and secure way to post is using the paid API (Option B).
> 
> * `postctl` attempts to minimize risk by imitating headers and inserting artificial delays (5 seconds between posts), but offers no guarantees.
> * X strictly requires the **`twid`** cookie (your User ID). Therefore, always enter the **entire cookie string** or both cookies (`auth_token` and `ct0`).

### Step 1: Extract Full Browser Cookies

The easiest way is to copy the entire Cookie header of any request:

1. Open your web browser, go to [x.com](https://x.com), and make sure you are logged in. (We recommend logging out and back in once to refresh the session).
2. Open Developer Tools (**F12** or **Cmd + Option + I**).
3. Switch to the **Network** tab.
4. Refresh the page (**F5** or **Cmd + R**).
5. Click on any request to `x.com` in the network request list (e.g., `home` or a GraphQL request).
6. In the right pane under **Request Headers**, look for the **`cookie:`** line.
7. Copy the **entire long value** (it usually starts with `guest_id=...` or `kdt=...` and contains all cookies).
8. Additionally, locate the value for **`ct0`** (your CSRF token, ~160 chars long) in the cookies and copy it as well.

### Step 2: Set up in `postctl`

We recommend setting it up via a one-liner in your terminal. Replace the placeholders with your copied values:

```bash
./postctl config setup twitter --cookie "YOUR_ENTIRE_COPIED_COOKIE_STRING" --ct0 "YOUR_ONLY_CT0_VALUE"
```

*(Note: If you start the interactive setup via `./postctl config setup twitter` and choose option `2`, you can also paste the entire long cookie string when prompted for the `auth_token`).*

### 🛠️ Troubleshooting Cookie Errors / Automation Warnings

If a GraphQL error like `empty tweet ID returned...` or error **226** (`This request looks like it might be automated...`) occurs when publishing a tweet, `postctl` automatically falls back to a **headless browser flow**:
1. **Automatic Browser Start**: `postctl` silently launches Google Chrome in the background (using `chromedp`), loads your cookies (`auth_token` & `ct0`), navigates to the X web interface, populates the composer (including threads and media uploads), and clicks "Post".
2. **Prerequisite**: Google Chrome must be installed on your system (searched by default in `/Applications` on macOS).
3. **Expired Session Cookies**: If the headless browser also fails, your cookies have likely expired. Simply repeat **Step 1** and enter the new cookies.

---

## Option B: Official API (Paid & Recommended)

If you own an official developer account and are willing to pay the monthly API costs, you can use the standard OAuth authentication.

> [!IMPORTANT]
> **Cost Warning (as of 2026):**
> Twitter/X no longer offers free write access (Free Tier) for newly created developer accounts. To post via the official interface, a paid API access (e.g., Basic Tier for ~$100/month or prepaid credits) is required in the Developer Portal.

### Step 1: Configure App in the Developer Portal
1. Go to the [Twitter Developer Portal](https://developer.twitter.com) and log in.
2. Create a new **Project** and a new **App** in your portal dashboard.
3. In the **App Settings**, navigate to **User authentication settings** and click **Set up**:
   * **App Type**: Choose **Web App, Automated App or Bot**.
   * **App Permissions**: Choose **Read and Write** (critical for posting rights).
   * **Type of App**: Choose **Native App** or **Single Page App** (for OAuth 2.0 PKCE).
   * **Callback URI / Redirect URL**: Enter exactly `http://localhost:8753/callback`.
   * **Website URL**: Enter your own website or `https://github.com/aeon022/postctl`.
4. Save the settings and copy the displayed **Client ID** and **Client Secret** to a secure place.

### Step 2: Store Credentials
```bash
# Set Client ID
./postctl config set twitter.client_id "YOUR_CLIENT_ID"

# Set Client Secret
./postctl config set twitter.client_secret "YOUR_CLIENT_SECRET"
```

### Step 3: Perform Authentication
```bash
./postctl auth twitter
```
A browser window will open asking you to authorize the app. After a successful login, `postctl` will store your encrypted access and refresh tokens.
