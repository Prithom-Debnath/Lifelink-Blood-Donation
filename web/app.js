/**
 * LifeLink — Simple, Beginner-Friendly Browser JavaScript
 * Communicates with the Go backend via standard fetch() API.
 * Demonstrates:
 * 1. User Account Flow (Registration, Login, Token Management, Profile View)
 * 2. Volunteer Blood Donor Management & Registration
 * 3. Blood Request Creation & Listing with Urgency Levels
 * 4. Emergency Compatibility Matching by Typed Location
 * 5. Text-based location search without external map services
 */

const TOKEN_KEY = 'lifelink_jwt_token';
let currentUser = null;
let currentAppRoute = 'sign-in';
let currentAppPage = 'account';

// Helper: Escape HTML characters to prevent XSS when inserting dynamic text into the DOM
function escapeHTML(str) {
  if (str === null || str === undefined) return '';
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

// Helper: Show feedback message (success or error) in a specific container
function showFeedback(elementId, message, isError) {
  const el = document.getElementById(elementId);
  if (!el) return;
  el.textContent = message;
  el.className = isError ? 'feedback-msg error' : 'feedback-msg success';
  el.style.display = 'block';

  // Automatically clear message after 6 seconds
  setTimeout(() => {
    el.style.display = 'none';
  }, 6000);
}

// Helper: JWT token retrieval, storage, and removal
function getToken() {
  return localStorage.getItem(TOKEN_KEY);
}

function setToken(token) {
  if (token) {
    localStorage.setItem(TOKEN_KEY, token);
  }
}

function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}

// -------------------------------------------------------------
// 2. USER ACCOUNT FLOW (REGISTER, LOGIN, PROFILE, LOGOUT)
// -------------------------------------------------------------

function animatePageEntry(element) {
  element.classList.remove('page-enter');
  void element.offsetWidth;
  element.classList.add('page-enter');
  element.addEventListener('animationend', () => {
    element.classList.remove('page-enter');
  }, { once: true });
}

function updateAppPageFromHash() {
  const requestedRoute = window.location.hash.slice(1) || 'sign-in';
  const navigationLinks = Array.from(document.querySelectorAll('#top-navigation [data-page-route]'));
  const selectedLink = navigationLinks.find((link) => link.dataset.pageRoute === requestedRoute)
    || navigationLinks.find((link) => link.dataset.pageRoute === 'sign-in');
  if (!selectedLink) return;

  const routeChanged = selectedLink.dataset.pageRoute !== currentAppRoute;
  const pageChanged = selectedLink.dataset.targetPage !== currentAppPage;
  currentAppRoute = selectedLink.dataset.pageRoute;
  currentAppPage = selectedLink.dataset.targetPage;

  document.querySelectorAll('main [data-page-view]').forEach((view) => {
    const shouldShow = view.dataset.pageView === currentAppPage;
    if (shouldShow) {
      const wasHidden = view.hidden;
      view.hidden = false;
      if (pageChanged && wasHidden) animatePageEntry(view);
    } else {
      view.hidden = true;
      view.classList.remove('page-enter');
    }
  });

  navigationLinks.forEach((link) => {
    const isCurrentRoute = link === selectedLink;
    link.classList.toggle('active', isCurrentRoute);
    if (isCurrentRoute) {
      link.setAttribute('aria-current', 'page');
    } else {
      link.removeAttribute('aria-current');
    }
  });

  if (currentAppPage === 'account') {
    switchAuthTab(selectedLink.dataset.authTab || 'login', routeChanged && !pageChanged);
  }
}
function navigateToAppRoute(route) {
  const nextHash = `#${route}`;
  if (window.location.hash === nextHash) {
    updateAppPageFromHash();
    return;
  }
  window.location.hash = route;
}

function switchAuthTab(tabName, animate = false) {
  const tabs = ['login', 'register', 'profile'];
  tabs.forEach(name => {
    const pane = document.getElementById(`pane-${name}`);
    if (!pane) return;

    if (name === tabName) {
      pane.style.display = 'block';
      pane.classList.add('active');
      if (animate) animatePageEntry(pane);
    } else {
      pane.style.display = 'none';
      pane.classList.remove('active');
    }
  });

  if (tabName === 'profile') {
    const hasProfile = Boolean(currentUser);
    const profileCard = document.getElementById('profile-card');
    const loginMessage = document.getElementById('profile-login-required');
    if (profileCard) profileCard.hidden = !hasProfile;
    if (loginMessage) loginMessage.hidden = hasProfile;
  }
}

function renderProfile(user) {
  currentUser = user;

  document.getElementById('profile-name').textContent = user.name || 'User';
  document.getElementById('profile-email').textContent = user.email || '—';
  document.getElementById('profile-id').textContent = user.id || '—';

  if (user.created_at) {
    const createdDate = new Date(user.created_at);
    document.getElementById('profile-created').textContent = createdDate.toLocaleDateString(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  } else {
    document.getElementById('profile-created').textContent = 'Just now';
  }

  const statusEl = document.getElementById('header-user-status');
  if (statusEl) {
    statusEl.innerHTML = `Signed in as&nbsp;<strong>${escapeHTML(user.name)}</strong>`;
    statusEl.hidden = false;
  }

  const authBtn = document.getElementById('header-auth-btn');
  if (authBtn) {
    authBtn.textContent = 'View Profile';
  }

  const badgeEl = document.getElementById('auth-badge');
  if (badgeEl) {
    badgeEl.textContent = 'Signed In';
    badgeEl.className = 'badge badge-available';
  }

  if (currentAppRoute === 'my-profile') switchAuthTab('profile');
}

function setLoggedOutState() {
  currentUser = null;
  clearToken();

  const statusEl = document.getElementById('header-user-status');
  if (statusEl) {
    statusEl.textContent = '';
    statusEl.hidden = true;
  }

  const authBtn = document.getElementById('header-auth-btn');
  if (authBtn) {
    authBtn.textContent = 'Sign In / Register';
  }

  const badgeEl = document.getElementById('auth-badge');
  if (badgeEl) {
    badgeEl.textContent = 'Guest';
    badgeEl.className = 'badge badge-guest';
  }

  if (currentAppRoute === 'my-profile') {
    switchAuthTab('profile');
  } else if (currentAppRoute === 'sign-in') {
    switchAuthTab('login');
  }
}

async function checkAuthAndLoadProfile() {
  const token = getToken();
  if (!token) {
    setLoggedOutState();
    return;
  }

  try {
    const res = await fetch('/api/auth/me', {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });

    if (!res.ok) {
      setLoggedOutState();
      return;
    }

    const user = await res.json();
    renderProfile(user);
  } catch (err) {
    console.warn('Could not verify authentication session:', err);
    setLoggedOutState();
  }
}

async function handleRegisterSubmit(e) {
  e.preventDefault();

  const name = document.getElementById('reg-name').value.trim();
  const email = document.getElementById('reg-email').value.trim();
  const password = document.getElementById('reg-password').value;

  if (password.length < 6) {
    showFeedback('register-feedback', 'Password must be at least 6 characters long.', true);
    return;
  }

  const payload = { name, email, password };

  try {
    const res = await fetch('/api/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });

    const data = await res.json();

    if (!res.ok) {
      showFeedback('register-feedback', data.error || 'Registration failed.', true);
      return;
    }

    setToken(data.token);
    document.getElementById('register-form').reset();
    renderProfile(data.user);
    navigateToAppRoute('my-profile');
    showFeedback('profile-feedback', `Account created successfully! Welcome to LifeLink, ${data.user.name}.`, false);
  } catch (err) {
    showFeedback('register-feedback', `Network error: ${err.message}`, true);
  }
}

async function handleLoginSubmit(e) {
  e.preventDefault();

  const email = document.getElementById('login-email').value.trim();
  const password = document.getElementById('login-password').value;

  const payload = { email, password };

  try {
    const res = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });

    const data = await res.json();

    if (!res.ok) {
      showFeedback('login-feedback', data.error || 'Login failed.', true);
      return;
    }

    setToken(data.token);
    document.getElementById('login-form').reset();
    renderProfile(data.user);
    navigateToAppRoute('my-profile');
    showFeedback('profile-feedback', `Signed in successfully. Welcome back, ${data.user.name}!`, false);
  } catch (err) {
    showFeedback('login-feedback', `Network error: ${err.message}`, true);
  }
}

function handleLogout() {
  setLoggedOutState();
  navigateToAppRoute('sign-in');
  showFeedback('login-feedback', 'You have been logged out cleanly.', false);
}

function handleHeaderAuthClick() {
  navigateToAppRoute(currentUser ? 'my-profile' : 'sign-in');
}

// -------------------------------------------------------------
// 3. VOLUNTEER DONOR LISTING & REGISTRATION
// -------------------------------------------------------------

async function loadDonors() {
  const tbody = document.getElementById('donors-tbody');
  tbody.innerHTML = '<tr><td colspan="5" class="text-center text-muted">Loading donors...</td></tr>';

  try {
    const res = await fetch('/api/donors');
    if (!res.ok) {
      throw new Error(`Server returned HTTP ${res.status}`);
    }
    const donors = await res.json();

    if (!Array.isArray(donors) || donors.length === 0) {
      tbody.innerHTML = '<tr><td colspan="5" class="text-center text-muted">No volunteer donors registered yet.</td></tr>';
      return;
    }

    tbody.innerHTML = donors.map(d => {
      return `
        <tr>
          <td><strong>${escapeHTML(d.name)}</strong></td>
          <td><span class="badge badge-blood">${escapeHTML(d.blood_group)}</span></td>
          <td>
            ${escapeHTML(d.city)}

          </td>
          <td>
            <div>${escapeHTML(d.phone)}</div>
            <small class="text-muted">${escapeHTML(d.email)}</small>
          </td>
          <td>
            <span class="badge ${d.availability ? 'badge-available' : 'badge-unavailable'}">
              ${d.availability ? 'Available' : 'Unavailable'}
            </span>
          </td>
        </tr>
      `;
    }).join('');
  } catch (err) {
    tbody.innerHTML = `<tr><td colspan="5" class="text-center text-muted">Failed to load donors: ${escapeHTML(err.message)}</td></tr>`;
  }
}

async function handleDonorSubmit(e) {
  e.preventDefault();

  const payload = {
    name: document.getElementById('donor-name').value.trim(),
    email: document.getElementById('donor-email').value.trim(),
    phone: document.getElementById('donor-phone').value.trim(),
    blood_group: document.getElementById('donor-blood-group').value,
    city: document.getElementById('donor-city').value.trim(),
    availability: document.getElementById('donor-availability').checked,
  };

  try {
    const res = await fetch('/api/donors', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });

    const data = await res.json();

    if (!res.ok) {
      showFeedback('donor-form-feedback', data.error || 'Failed to register volunteer donor.', true);
      return;
    }

    showFeedback('donor-form-feedback', `Volunteer donor ${data.name} (${data.blood_group}) registered successfully!`, false);
    document.getElementById('donor-form').reset();
    document.getElementById('donor-availability').checked = true;

    loadDonors();
  } catch (err) {
    showFeedback('donor-form-feedback', `Network error: ${err.message}`, true);
  }
}

// -------------------------------------------------------------
// 4. BLOOD REQUEST LISTING & CREATION (WITH URGENCY)
// -------------------------------------------------------------

async function loadBloodRequests() {
  const tbody = document.getElementById('requests-tbody');
  tbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted">Loading requests...</td></tr>';

  const urgencyFilter = document.getElementById('filter-urgency').value;
  const sortBy = document.getElementById('sort-requests').value;

  let url = '/api/requests?';
  const params = [];
  if (urgencyFilter) {
    params.push(`urgency=${encodeURIComponent(urgencyFilter)}`);
  }
  if (sortBy) {
    params.push(`sort=${encodeURIComponent(sortBy)}`);
  }
  url += params.join('&');

  try {
    const res = await fetch(url);
    if (!res.ok) {
      throw new Error(`Server returned HTTP ${res.status}`);
    }
    const requests = await res.json();

    if (!Array.isArray(requests) || requests.length === 0) {
      tbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted">No blood requests found.</td></tr>';
      return;
    }

    tbody.innerHTML = requests.map(r => {
      let statusBadge = 'badge-open';
      if (r.status === 'fulfilled') statusBadge = 'badge-fulfilled';
      if (r.status === 'cancelled') statusBadge = 'badge-cancelled';

      const urgency = (r.urgency || 'normal').toLowerCase();
      let urgencyBadge = 'badge-normal';
      if (urgency === 'critical') urgencyBadge = 'badge-critical';
      else if (urgency === 'urgent') urgencyBadge = 'badge-urgent';

      return `
        <tr>
          <td><strong>${escapeHTML(r.patient_name)}</strong></td>
          <td><span class="badge badge-blood">${escapeHTML(r.blood_group)}</span></td>
          <td><span class="badge ${urgencyBadge}">${escapeHTML(urgency.toUpperCase())}</span></td>
          <td>
            <div>${escapeHTML(r.hospital)}</div>
            <small class="text-muted">${escapeHTML(r.city)}</small>

          </td>
          <td>${escapeHTML(r.contact_phone)}</td>
          <td><span class="badge ${statusBadge}">${escapeHTML(r.status)}</span></td>
        </tr>
      `;
    }).join('');
  } catch (err) {
    tbody.innerHTML = `<tr><td colspan="6" class="text-center text-muted">Failed to load requests: ${escapeHTML(err.message)}</td></tr>`;
  }
}

async function handleRequestSubmit(e) {
  e.preventDefault();

  const payload = {
    patient_name: document.getElementById('req-patient-name').value.trim(),
    blood_group: document.getElementById('req-blood-group').value,
    urgency: document.getElementById('req-urgency').value,
    hospital: document.getElementById('req-hospital').value.trim(),
    city: document.getElementById('req-city').value.trim(),
    contact_phone: document.getElementById('req-contact-phone').value.trim(),
    status: document.getElementById('req-status').value,
  };

  try {
    const res = await fetch('/api/requests', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });

    const data = await res.json();

    if (!res.ok) {
      showFeedback('request-form-feedback', data.error || 'Failed to submit blood request.', true);
      return;
    }

    showFeedback('request-form-feedback', `Request for ${data.patient_name} (${(data.urgency || 'normal').toUpperCase()}) submitted successfully!`, false);
    document.getElementById('request-form').reset();
    document.getElementById('req-status').value = 'open';
    document.getElementById('req-urgency').value = 'normal';

    loadBloodRequests();
  } catch (err) {
    showFeedback('request-form-feedback', `Network error: ${err.message}`, true);
  }
}

// -------------------------------------------------------------
// 5. EMERGENCY DONOR COMPATIBILITY MATCHING SEARCH
// -------------------------------------------------------------

async function handleMatchSearch(e) {
  e.preventDefault();

  const bloodGroup = document.getElementById('match-blood-group').value;
  const city = document.getElementById('match-city').value.trim();
  const tbody = document.getElementById('matches-tbody');

  if (!bloodGroup) {
    showFeedback('match-feedback', 'Please select a recipient blood group.', true);
    return;
  }

  tbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted">Searching for compatible donors...</td></tr>';

  let url = `/api/matches?bloodGroup=${encodeURIComponent(bloodGroup)}`;
  if (city) {
    url += `&city=${encodeURIComponent(city)}`;
  }

  try {
    const res = await fetch(url);
    const data = await res.json();

    if (!res.ok) {
      showFeedback('match-feedback', data.error || 'Failed to search matches.', true);
      tbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted">Search error.</td></tr>';
      return;
    }

    if (!Array.isArray(data) || data.length === 0) {
      const locationText = city ? ` near ${escapeHTML(city)}` : '';
      tbody.innerHTML = `<tr><td colspan="6" class="text-center text-muted">No available donors found compatible with recipient <strong>${escapeHTML(bloodGroup)}</strong>${locationText}.</td></tr>`;
      showFeedback('match-feedback', 'No matching donors found.', false);
      return;
    }

    tbody.innerHTML = data.map(d => `
      <tr>
        <td><strong>${escapeHTML(d.name)}</strong></td>
        <td><span class="badge badge-blood">${escapeHTML(d.blood_group)}</span></td>
        <td>${escapeHTML(d.city)}</td>
        <td>${escapeHTML(d.phone)}</td>
        <td>${escapeHTML(d.email)}</td>
        <td>
          <span class="badge ${d.availability ? 'badge-available' : 'badge-unavailable'}">
            ${d.availability ? 'Available' : 'Unavailable'}
          </span>
        </td>
      </tr>
    `).join('');

    showFeedback('match-feedback', `Found ${data.length} compatible available donor(s)!`, false);
  } catch (err) {
    showFeedback('match-feedback', `Network error: ${err.message}`, true);
    tbody.innerHTML = `<tr><td colspan="6" class="text-center text-muted">Error: ${escapeHTML(err.message)}</td></tr>`;
  }
}
// -------------------------------------------------------------
// 6. INITIALIZE EVENT LISTENERS ON PAGE LOAD
// -------------------------------------------------------------
document.addEventListener('DOMContentLoaded', () => {
  // Main navigation swaps the visible page and records the selected page in the URL hash.
  window.addEventListener('hashchange', updateAppPageFromHash);
  document.getElementById('top-navigation').addEventListener('click', (event) => {
    const link = event.target.closest('[data-page-route]');
    if (link && window.location.hash === link.hash) {
      event.preventDefault();
      updateAppPageFromHash();
    }
  });
  if (!window.location.hash) {
    window.history.replaceState(null, '', '#sign-in');
  }
  updateAppPageFromHash();

  // Bind User Account forms and actions
  document.getElementById('login-form').addEventListener('submit', handleLoginSubmit);
  document.getElementById('register-form').addEventListener('submit', handleRegisterSubmit);
  document.getElementById('logout-btn').addEventListener('click', handleLogout);
  document.getElementById('header-auth-btn').addEventListener('click', handleHeaderAuthClick);

  // Bind Donor, Request, and Match forms
  document.getElementById('donor-form').addEventListener('submit', handleDonorSubmit);
  document.getElementById('request-form').addEventListener('submit', handleRequestSubmit);
  document.getElementById('match-form').addEventListener('submit', handleMatchSearch);

  // Bind Blood Request filtering and sorting toolbar
  document.getElementById('filter-urgency').addEventListener('change', loadBloodRequests);
  document.getElementById('sort-requests').addEventListener('change', loadBloodRequests);

  // Bind refresh buttons
  document.getElementById('refresh-donors-btn').addEventListener('click', loadDonors);
  document.getElementById('refresh-requests-btn').addEventListener('click', loadBloodRequests);

  // Check authentication session and load initial data
  checkAuthAndLoadProfile();
  loadDonors();
  loadBloodRequests();

});
