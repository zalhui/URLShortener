const state = {
  accessToken: null,
  user: null,
  links: [],
  authMode: "login",
};

const elements = {
  flash: document.querySelector("#flash"),
  authView: document.querySelector("#auth-view"),
  dashboardView: document.querySelector("#dashboard-view"),
  loginTab: document.querySelector("#login-tab"),
  registerTab: document.querySelector("#register-tab"),
  loginForm: document.querySelector("#login-form"),
  registerForm: document.querySelector("#register-form"),
  loginEmail: document.querySelector("#login-email"),
  loginPassword: document.querySelector("#login-password"),
  registerEmail: document.querySelector("#register-email"),
  registerPassword: document.querySelector("#register-password"),
  loginSubmit: document.querySelector("#login-submit"),
  registerSubmit: document.querySelector("#register-submit"),
  userEmail: document.querySelector("#user-email"),
  linksCount: document.querySelector("#links-count"),
  dashboardTitle: document.querySelector("#dashboard-title"),
  dashboardSubtitle: document.querySelector("#dashboard-subtitle"),
  createForm: document.querySelector("#create-form"),
  createURL: document.querySelector("#create-url"),
  createSubmit: document.querySelector("#create-submit"),
  createResult: document.querySelector("#create-result"),
  refreshLinks: document.querySelector("#refresh-links"),
  logoutButton: document.querySelector("#logout-button"),
  emptyState: document.querySelector("#empty-state"),
  linksList: document.querySelector("#links-list"),
};

document.addEventListener("DOMContentLoaded", () => {
  bindEvents();
  render();
  bootstrapSession();
});

function bindEvents() {
  elements.loginTab.addEventListener("click", () => setAuthMode("login"));
  elements.registerTab.addEventListener("click", () => setAuthMode("register"));

  elements.loginForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    await handleAuthSubmit("/auth/login", {
      email: elements.loginEmail.value,
      password: elements.loginPassword.value,
    }, elements.loginSubmit, "Signed in.");
  });

  elements.registerForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    await handleAuthSubmit("/auth/register", {
      email: elements.registerEmail.value,
      password: elements.registerPassword.value,
    }, elements.registerSubmit, "Account created.");
  });

  elements.createForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    await createLink();
  });

  elements.refreshLinks.addEventListener("click", async () => {
    await fetchLinks();
  });

  elements.logoutButton.addEventListener("click", async () => {
    await logout();
  });

  elements.linksList.addEventListener("click", async (event) => {
    const copyButton = event.target.closest("[data-copy-url]");
    if (copyButton) {
      await copyToClipboard(copyButton.dataset.copyUrl);
      return;
    }

    const deleteButton = event.target.closest("[data-delete-id]");
    if (deleteButton) {
      await deleteLink(deleteButton.dataset.deleteId, deleteButton.dataset.deleteLabel);
    }
  });
}

function setAuthMode(mode) {
  state.authMode = mode;
  render();
}

async function bootstrapSession() {
  const restored = await refreshSession(true);
  if (!restored) {
    showFlash("Sign in to manage your links. Public short URLs still work for anyone.", "success");
    return;
  }

  showFlash(`Session restored for ${state.user.email}.`, "success");
  await fetchLinks();
}

async function handleAuthSubmit(path, payload, button, successMessage) {
  setButtonBusy(button, true, "Working...");

  try {
    const response = await fetch(path, {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
    });

    const result = await parseJSON(response);
    if (!response.ok) {
      throw new Error(readErrorMessage(response, result));
    }

    applySession(result);
    resetAuthForms();
    render();
    showFlash(successMessage, "success");
    await fetchLinks();
  } catch (error) {
    showFlash(error.message, "error");
  } finally {
    setButtonBusy(button, false);
  }
}

async function refreshSession(silent = false) {
  try {
    const response = await fetch("/auth/refresh", {
      method: "POST",
      credentials: "include",
    });

    if (!response.ok) {
      clearSession();
      if (!silent) {
        showFlash("Session expired. Sign in again.", "error");
      }
      return false;
    }

    const result = await parseJSON(response);
    applySession(result);
    render();
    return true;
  } catch (error) {
    clearSession();
    if (!silent) {
      showFlash("Failed to restore session.", "error");
    }
    return false;
  }
}

async function fetchLinks() {
  try {
    const response = await authorizedFetch("/api/urls", {
      method: "GET",
    });
    const result = await parseJSON(response);
    state.links = Array.isArray(result) ? result : [];
    renderLinks();
  } catch (error) {
    showFlash(error.message, "error");
  }
}

async function createLink() {
  const originalURL = elements.createURL.value.trim();
  if (!originalURL) {
    showFlash("Enter a URL to shorten.", "error");
    return;
  }

  setButtonBusy(elements.createSubmit, true, "Creating...");

  try {
    const response = await authorizedFetch("/api/shorten", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ url: originalURL }),
    });
    const result = await parseJSON(response);

    elements.createForm.reset();
    renderCreateResult(toPublicShortURL(result.result));
    showFlash("Short link created.", "success");
    await fetchLinks();
  } catch (error) {
    showFlash(error.message, "error");
  } finally {
    setButtonBusy(elements.createSubmit, false);
  }
}

async function deleteLink(shortID, shortURL) {
  if (!shortID) {
    return;
  }

  try {
    const response = await authorizedFetch(`/api/urls/${encodeURIComponent(shortID)}`, {
      method: "DELETE",
    });

    if (!response.ok) {
      const body = await response.text();
      throw new Error(body || "Failed to delete link.");
    }

    showFlash(`Deleted ${shortURL}.`, "success");
    state.links = state.links.filter((item) => item.id !== shortID);
    renderLinks();
  } catch (error) {
    showFlash(error.message, "error");
  }
}

async function logout() {
  try {
    await fetch("/auth/logout", {
      method: "POST",
      credentials: "include",
    });
  } finally {
    clearSession();
    elements.createResult.classList.add("hidden");
    render();
    showFlash("Signed out.", "success");
  }
}

async function authorizedFetch(url, options = {}, retry = true) {
  if (!state.accessToken) {
    const restored = await refreshSession(true);
    if (!restored) {
      throw new Error("Authentication required.");
    }
  }

  const response = await fetch(url, {
    ...options,
    headers: {
      ...(options.headers || {}),
      Authorization: `Bearer ${state.accessToken}`,
    },
  });

  if (response.status === 401 && retry) {
    const restored = await refreshSession(true);
    if (!restored) {
      throw new Error("Authentication required.");
    }

    return authorizedFetch(url, options, false);
  }

  if (!response.ok) {
    const body = await response.text();
    throw new Error(body || "Request failed.");
  }

  return response;
}

function applySession(result) {
  state.accessToken = result.access_token;
  state.user = result.user;
}

function clearSession() {
  state.accessToken = null;
  state.user = null;
  state.links = [];
}

function render() {
  const authenticated = Boolean(state.user && state.accessToken);

  elements.authView.classList.toggle("hidden", authenticated);
  elements.dashboardView.classList.toggle("hidden", !authenticated);

  elements.loginTab.classList.toggle("active", state.authMode === "login");
  elements.registerTab.classList.toggle("active", state.authMode === "register");
  elements.loginForm.classList.toggle("hidden", state.authMode !== "login");
  elements.registerForm.classList.toggle("hidden", state.authMode !== "register");

  if (!authenticated) {
    elements.linksList.innerHTML = "";
    elements.emptyState.classList.add("hidden");
    return;
  }

  elements.userEmail.textContent = state.user.email;
  elements.dashboardTitle.textContent = `Links for ${state.user.email}`;
  elements.dashboardSubtitle.textContent = "Create, copy and delete only the URLs you own.";
  renderLinks();
}

function renderLinks() {
  const links = state.links || [];
  elements.linksCount.textContent = String(links.length);
  elements.linksList.innerHTML = "";
  elements.emptyState.classList.toggle("hidden", links.length > 0);

  if (links.length === 0) {
    return;
  }

  const fragment = document.createDocumentFragment();
  for (const item of links) {
    const li = document.createElement("li");
    li.className = "link-card";
    li.innerHTML = `
      <div class="link-head">
        <div>
          <a class="link-title" href="${escapeHTML(toPublicShortURL(item.short_url))}" target="_blank" rel="noreferrer">${escapeHTML(toPublicShortURL(item.short_url))}</a>
          <a class="link-original" href="${escapeHTML(item.original_url)}" target="_blank" rel="noreferrer">${escapeHTML(item.original_url)}</a>
        </div>
        <div class="link-actions">
          <button class="link-action" type="button" data-copy-url="${escapeAttribute(toPublicShortURL(item.short_url))}">Copy</button>
          <button class="link-action delete" type="button" data-delete-id="${escapeAttribute(item.id)}" data-delete-label="${escapeAttribute(toPublicShortURL(item.short_url))}">Delete</button>
        </div>
      </div>
      <div class="link-meta">
        <span>ID: ${escapeHTML(item.id)}</span>
        <span>Created: ${formatDate(item.created_at)}</span>
      </div>
    `;
    fragment.appendChild(li);
  }

  elements.linksList.appendChild(fragment);
}

function renderCreateResult(shortURL) {
  elements.createResult.innerHTML = `
    <strong>Short URL ready:</strong>
    <div><a href="${escapeHTML(shortURL)}" target="_blank" rel="noreferrer">${escapeHTML(shortURL)}</a></div>
  `;
  elements.createResult.classList.remove("hidden");
}

function toPublicShortURL(value) {
  try {
    const parsed = new URL(value, window.location.origin);
    return `${window.location.origin}${parsed.pathname}`;
  } catch (error) {
    return value;
  }
}

function resetAuthForms() {
  elements.loginForm.reset();
  elements.registerForm.reset();
}

function setButtonBusy(button, busy, busyLabel) {
  if (!button.dataset.defaultLabel) {
    button.dataset.defaultLabel = button.textContent;
  }

  button.disabled = busy;
  button.textContent = busy ? busyLabel : button.dataset.defaultLabel;
}

function showFlash(message, type) {
  elements.flash.textContent = message;
  elements.flash.className = `flash ${type}`;
}

function readErrorMessage(response, result) {
  if (result && typeof result.error === "string" && result.error) {
    return result.error;
  }
  if (typeof result === "string" && result) {
    return result;
  }
  return response.statusText || "Request failed.";
}

async function parseJSON(response) {
  const contentType = response.headers.get("Content-Type") || "";
  if (!contentType.includes("application/json")) {
    return response.text();
  }
  return response.json();
}

async function copyToClipboard(value) {
  try {
    await navigator.clipboard.writeText(value);
    showFlash("Short URL copied to clipboard.", "success");
  } catch (error) {
    showFlash("Clipboard access is unavailable.", "error");
  }
}

function formatDate(value) {
  if (!value) {
    return "unknown";
  }

  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(parsed);
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function escapeAttribute(value) {
  return escapeHTML(value);
}
