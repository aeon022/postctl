# Threads (Meta) API Setup Guide for postctl

This document describes in detail how to obtain access to the official Threads API, configure your app in the Meta Developer Portal, and set up authentication for `postctl`.

---

## 1. Create a Meta Developer App

1. Go to the [Meta Developer Portal](https://developers.facebook.com) and log in.
2. Click on **My Apps** in the top right corner and then **Create App**:
   * Choose **Other** as the app type and click *Next*.
   * Choose **Consumer** (or any type that allows access to the Threads/Instagram API).
   * Enter a name for your application (e.g., `postctl-threads-app`).
   * Click **Create App** (you may be prompted to enter your Facebook password).

---

## 2. Add Threads API

1. In your app dashboard, scroll down to **Add Products to Your App**.
2. Find **Threads API** and click **Set Up**.
3. Your app is now in **In Development** mode – this is perfect and important for local development.

---

## 3. Configure Callback URLs (Important: Mandatory Fields & HTTPS!)

Meta strictly enforces secure HTTPS connections for the Threads API. Furthermore, the dashboard will refuse to save changes if deinstallation or deletion callback URLs are missing.

1. In the left navigation bar, under **Threads API**, click **Threads Settings** (or go to **Use Cases** ➔ **Access Threads API** ➔ **Customize / Settings**).
2. Configure the following three fields:
   * **Valid OAuth Redirect URIs**:
     `https://localhost:8753/callback` (Important: **https** instead of http!)
   * **Uninstall Callback URL**:
     `https://localhost:8753/uninstall` (or `https://example.com/uninstall` as a placeholder)
   * **Delete Callback URL**:
     `https://localhost:8753/delete` (or `https://example.com/delete` as a placeholder)
3. Click **Save** in the bottom right.

---

## 4. App Roles & Tester Invitation (Important!)

Since your app is in development mode, it can only be authenticated by users registered as testers.

### Step A: Add Tester in the Meta Dashboard
1. Click **App Roles** (the people icon) ➔ **Roles** in the left navigation bar.
2. Scroll to the bottom to the **Threads Testers** section.
3. Click **Add Threads Testers**.
4. Enter the exact Instagram/Threads username of the account you want to post from, and click **Confirm**.

### Step B: Accept Invitation on Instagram
The invitation must be confirmed manually from your Threads account:
1. Log in to **[Instagram.com](https://www.instagram.com/)** in a desktop browser using the corresponding account.
2. Go to your profile ➔ Click the **Gear icon** (Settings).
3. Navigate to **Apps and Websites** in the left menu.
4. Click on the **Tester Invites** tab at the top.
5. You will see the invite from your app (e.g., `postctl-threads-app`). Click **Accept**.

---

## 5. Retrieve App ID & App Secret

1. In the left navigation bar, go to **App Settings** ➔ **Basic**.
2. Copy the **App ID**.
3. Copy the **App Secret** (requires clicking *Show* and entering your password).

---

## 6. Configure in `postctl`

Save your Threads credentials in your terminal (use `./postctl` for the local binary):

```bash
# Set App ID
./postctl config set threads.app_id "YOUR_APP_ID"

# Set App Secret
./postctl config set threads.app_secret "YOUR_APP_SECRET"
```

Verify your configuration:
```bash
./postctl config show
```

---

## 7. Establish Connection (Auth Flow)

Run the authentication flow:

```bash
./postctl auth threads
```

1. Your default browser will open and redirect you to the Threads login dialog.
2. Log in with your Threads account and grant permission to access your profile data and media (`threads_basic` and `threads_content_publish`).
3. After logging in, Meta redirects you back to `https://localhost:8753/callback`.
4. **Handling the SSL Warning in the Browser:**
   * Because `postctl`'s local server uses a temporary, self-signed SSL certificate for encryption, your browser will display a warning **"Your connection is not private"** (or `NET::ERR_CERT_AUTHORITY_INVALID`).
   * This is completely normal and safe, as all communication takes place locally on your Mac.
   * Click **"Advanced"** (or *Show Details*) in the browser window and select **"Proceed to localhost (unsafe)"**.
5. Once clicked, `postctl` receives the access token, exchanges it in the background for a long-lived token valid for 60 days, and stores it encrypted in the SQLite database.
6. A success message will appear in your terminal! You are now ready to post updates to Threads.
