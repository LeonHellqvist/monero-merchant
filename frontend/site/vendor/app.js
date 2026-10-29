// ============ Configuration ============
const API_BASE = '/api';

// ============ Clipboard (works over plain http where navigator.clipboard is unavailable) ============
function clipCopy(text) {
  try {
    if (navigator.clipboard && window.isSecureContext) { navigator.clipboard.writeText(text); return true; }
  } catch (e) {}
  try {
    const ta = document.createElement('textarea');
    ta.value = text; ta.setAttribute('readonly', '');
    ta.style.position = 'fixed'; ta.style.top = '-1000px'; ta.style.opacity = '0';
    document.body.appendChild(ta); ta.focus(); ta.select(); ta.setSelectionRange(0, ta.value.length);
    const ok = document.execCommand('copy'); document.body.removeChild(ta); return ok;
  } catch (e) { return false; }
}

// ============ Security: HTML Escaping ============
function escapeHtml(str) {
  if (str === null || str === undefined) return '';
  return String(str).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]);
}

// Errors come as JSON {error} / {message} or as plain text
async function errorMessage(res, fallback) {
  const text = await res.text().catch(() => '');
  try {
    const data = JSON.parse(text);
    if (data.error || data.message) return data.error || data.message;
  } catch (e) {}
  return text.trim() || fallback;
}

// ============ State ============
let auth = {
  accessToken: null,
  refreshToken: null,
  vendorName: null,
  vendorId: null
};

let currentPage = 'dashboard';
let transactions = { confirmed: [], pending: [] };

// ============ Utilities ============
const $ = id => document.getElementById(id);
const $$ = sel => document.querySelectorAll(sel);

function formatXMR(atomic) {
  if (!atomic && atomic !== 0) return '0.000000';
  const xmr = atomic / 1e12;
  return xmr.toLocaleString('en-US', { minimumFractionDigits: 6, maximumFractionDigits: 6 });
}

function formatDate(dateStr) {
  if (!dateStr) return '-';
  const d = new Date(dateStr);
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit' });
}

// ============ API Calls ============
async function api(endpoint, options = {}) {
  const headers = { 'Content-Type': 'application/json', ...options.headers };
  if (auth.accessToken) {
    headers['Authorization'] = 'Bearer ' + auth.accessToken;
  }

  const response = await fetch(API_BASE + endpoint, { ...options, headers });

  if (response.status === 401 && auth.refreshToken) {
    const refreshed = await refreshAccessToken();
    if (refreshed) {
      headers['Authorization'] = 'Bearer ' + auth.accessToken;
      return fetch(API_BASE + endpoint, { ...options, headers });
    }
  }

  return response;
}

async function refreshAccessToken() {
  try {
    const res = await fetch(API_BASE + '/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: auth.refreshToken })
    });
    if (res.ok) {
      const data = await res.json();
      auth.accessToken = data.access_token;
      auth.refreshToken = data.refresh_token || auth.refreshToken;
      saveAuth();
      return true;
    }
  } catch (e) {
    console.error('Token refresh failed:', e);
  }
  logout();
  return false;
}

// ============ Auth ============
// access token is kept in memory only, refresh token is persisted so the
// user stays logged in across reloads (it gets exchanged for a new access
// token on page load).
function saveAuth() {
  localStorage.setItem('mm_vendor_auth', JSON.stringify({
    refreshToken: auth.refreshToken,
    vendorName: auth.vendorName,
    vendorId: auth.vendorId
  }));
}

function loadAuth() {
  const saved = localStorage.getItem('mm_vendor_auth');
  if (!saved) return false;
  try {
    const data = JSON.parse(saved);
    auth.refreshToken = data.refreshToken || null;
    auth.vendorName = data.vendorName || null;
    auth.vendorId = data.vendorId != null ? data.vendorId : null;
    return auth.refreshToken !== null;
  } catch (e) {
    localStorage.removeItem('mm_vendor_auth');
    return false;
  }
}

function logout() {
  auth = { accessToken: null, refreshToken: null, vendorName: null, vendorId: null };
  localStorage.removeItem('mm_vendor_auth');
  location.replace(location.pathname);
}

async function login(username, password) {
  const res = await fetch(API_BASE + '/auth/login-vendor', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: username, password })
  });

  if (!res.ok) {
    throw new Error(await errorMessage(res, 'Invalid credentials'));
  }

  const data = await res.json();
  auth.accessToken = data.access_token;
  auth.refreshToken = data.refresh_token;
  auth.vendorName = username;
  auth.vendorId = data.vendor_id;
  saveAuth();
}

async function signup(username, email, password, address, inviteCode) {
  if (new TextEncoder().encode(email).length > 254) {
    throw new Error('Email must be no more than 254 bytes');
  }
  const res = await fetch(API_BASE + '/vendor/create', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      name: username,
      email: email || '',
      password,
      monero_subaddress: address,
      invite_code: inviteCode
    })
  });

  if (!res.ok) {
    throw new Error(await errorMessage(res, 'Could not create account'));
  }

  await login(username, password);
}

// ============ Data Fetching ============
async function fetchBalance() {
  const res = await api('/vendor/balance');
  if (res.ok) {
    return await res.json();
  }
  return { balance: 0 };
}

async function fetchTransactions() {
  const res = await api('/vendor/transactions');
  if (res.ok) {
    const data = await res.json();
    transactions.confirmed = data.confirmed_transactions || [];
    transactions.pending = data.pending_transactions || [];
    return transactions;
  }
  return { confirmed: [], pending: [] };
}

async function fetchPosDevices() {
  const res = await api('/vendor/pos-list');
  if (res.ok) {
    const data = await res.json();
    return data.devices || [];
  }
  return [];
}

async function initiateWithdrawal() {
  const res = await api('/vendor/transfer-balance', { method: 'POST' });
  if (!res.ok) {
    return { ok: false, error: await errorMessage(res, 'Failed to initiate withdrawal') };
  }
  const data = await res.json().catch(() => ({}));
  return { ok: true, message: data.message };
}

async function createPOS(name, password) {
  const res = await api('/vendor/create-pos', {
    method: 'POST',
    body: JSON.stringify({ name, password })
  });

  if (!res.ok) {
    throw new Error(await errorMessage(res, 'Could not create POS device'));
  }

  return await res.json();
}

async function exportCSV() {
  const res = await api('/vendor/export');
  if (res.ok) {
    const data = await res.json();
    const blob = new Blob([data.csv_data], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'monero-merchant-transactions-' + new Date().toISOString().split('T')[0] + '.csv';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  } else {
    alert(await errorMessage(res, 'No transactions to export'));
  }
}

// ============ DOM Builders (Safe) ============
function createStatCard(label, value, isBalance) {
  const card = document.createElement('div');
  card.className = 'stat-card';

  const labelEl = document.createElement('div');
  labelEl.className = 'stat-label';
  labelEl.textContent = label;

  const valueEl = document.createElement('div');
  valueEl.className = 'stat-value' + (isBalance ? ' balance' : '');
  valueEl.textContent = value;

  card.appendChild(labelEl);
  card.appendChild(valueEl);
  return card;
}

function createTransactionRow(tx) {
  const tr = document.createElement('tr');

  const tdId = document.createElement('td');
  tdId.textContent = '#' + tx.id;

  const tdAmount = document.createElement('td');
  tdAmount.className = 'text-mono';
  tdAmount.textContent = formatXMR(tx.amount) + ' XMR';

  const tdStatus = document.createElement('td');
  const badge = document.createElement('span');
  badge.className = 'status-badge';
  if (tx.transferred) {
    badge.className += ' status-transferred';
    badge.textContent = 'Transferred';
  } else if (tx.confirmed) {
    badge.className += ' status-confirmed';
    badge.textContent = 'Confirmed';
  } else if (tx.accepted) {
    badge.className += ' status-pending';
    badge.textContent = 'Pending';
  } else {
    badge.className += ' status-pending';
    badge.textContent = 'Awaiting Payment';
  }
  tdStatus.appendChild(badge);

  const tdPos = document.createElement('td');
  tdPos.textContent = tx.pos_name || '-';

  const tdDesc = document.createElement('td');
  tdDesc.textContent = tx.description || '-';

  const tdDate = document.createElement('td');
  tdDate.className = 'text-muted';
  tdDate.textContent = formatDate(tx.created_at);

  tr.appendChild(tdId);
  tr.appendChild(tdAmount);
  tr.appendChild(tdStatus);
  tr.appendChild(tdPos);
  tr.appendChild(tdDesc);
  tr.appendChild(tdDate);

  return tr;
}

function createTransactionTable(txList) {
  const wrapper = document.createElement('div');
  wrapper.className = 'table-wrapper';

  if (!txList || txList.length === 0) {
    const empty = document.createElement('div');
    empty.className = 'empty-state';
    empty.innerHTML = '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"/></svg>';
    const h3 = document.createElement('h3');
    h3.textContent = 'No transactions yet';
    const p = document.createElement('p');
    p.textContent = 'Transactions will appear here once payments are received';
    empty.appendChild(h3);
    empty.appendChild(p);
    wrapper.appendChild(empty);
    return wrapper;
  }

  const table = document.createElement('table');
  const thead = document.createElement('thead');
  const headerRow = document.createElement('tr');
  ['ID', 'Amount', 'Status', 'POS Device', 'Description', 'Date'].forEach(text => {
    const th = document.createElement('th');
    th.textContent = text;
    headerRow.appendChild(th);
  });
  thead.appendChild(headerRow);
  table.appendChild(thead);

  const tbody = document.createElement('tbody');
  txList.forEach(tx => {
    tbody.appendChild(createTransactionRow(tx));
  });
  table.appendChild(tbody);
  wrapper.appendChild(table);

  return wrapper;
}

// ============ UI Rendering ============
function showLoginScreen() {
  $('login-screen').classList.remove('hidden');
  $('app-screen').classList.add('hidden');
}

function showAppScreen() {
  $('login-screen').classList.add('hidden');
  $('app-screen').classList.remove('hidden');
  $('user-name').textContent = auth.vendorName || 'Vendor';
  $('user-avatar').textContent = (auth.vendorName || 'V')[0].toUpperCase();
  navigateTo('dashboard');
}

function navigateTo(page) {
  currentPage = page;

  $$('.nav-item').forEach(item => {
    item.classList.toggle('active', item.dataset.page === page);
  });

  const titles = {
    dashboard: 'Dashboard',
    transactions: 'Transactions',
    pos: 'POS Devices',
    settings: 'Settings'
  };
  $('page-title').textContent = titles[page] || 'Dashboard';

  renderPage(page);
}

async function renderPage(page) {
  const content = $('page-content');
  const headerActions = $('header-actions');

  while (content.firstChild) content.removeChild(content.firstChild);
  while (headerActions.firstChild) headerActions.removeChild(headerActions.firstChild);

  const loading = document.createElement('div');
  loading.className = 'loading';
  loading.innerHTML = '<div class="spinner"></div>';
  content.appendChild(loading);

  switch (page) {
    case 'dashboard':
      await renderDashboard(content, headerActions);
      break;
    case 'transactions':
      await renderTransactions(content, headerActions);
      break;
    case 'pos':
      await renderPOS(content, headerActions);
      break;
    case 'settings':
      renderSettings(content, headerActions);
      break;
  }
}

async function renderDashboard(container, headerActions) {
  const [balanceData] = await Promise.all([
    fetchBalance(),
    fetchTransactions()
  ]);

  const totalConfirmed = transactions.confirmed.length;
  const totalPending = transactions.pending.length;
  const todayTx = transactions.confirmed.filter(tx => {
    const d = new Date(tx.created_at);
    const today = new Date();
    return d.toDateString() === today.toDateString();
  }).length;

  while (container.firstChild) container.removeChild(container.firstChild);

  // Stats grid
  const statsGrid = document.createElement('div');
  statsGrid.className = 'stats-grid';
  statsGrid.appendChild(createStatCard('Available Balance', formatXMR(balanceData.balance) + ' XMR', true));
  statsGrid.appendChild(createStatCard('Pending Transactions', String(totalPending), false));
  statsGrid.appendChild(createStatCard('Confirmed Transactions', String(totalConfirmed), false));
  statsGrid.appendChild(createStatCard("Today's Transactions", String(todayTx), false));
  container.appendChild(statsGrid);

  // Withdrawal card
  const withdrawCard = document.createElement('div');
  withdrawCard.className = 'card';
  withdrawCard.style.marginBottom = '20px';

  const withdrawHeader = document.createElement('div');
  withdrawHeader.className = 'card-header';
  const withdrawTitle = document.createElement('h3');
  withdrawTitle.className = 'card-title';
  withdrawTitle.textContent = 'Withdraw Funds';
  withdrawHeader.appendChild(withdrawTitle);
  withdrawCard.appendChild(withdrawHeader);

  const withdrawInfo = document.createElement('p');
  withdrawInfo.className = 'text-muted';
  withdrawInfo.textContent = 'Transfer your confirmed balance to your Monero payout address. Minimum withdrawal: 0.003 XMR. Transfers are processed automatically within 30 seconds.';
  withdrawCard.appendChild(withdrawInfo);

  const withdrawBtnDiv = document.createElement('div');
  withdrawBtnDiv.className = 'mt-20';
  const withdrawBtn = document.createElement('button');
  withdrawBtn.className = 'btn btn-primary';
  withdrawBtn.textContent = 'Initiate Withdrawal';
  withdrawBtn.disabled = balanceData.balance < 3000000000;
  if (balanceData.balance < 3000000000) {
    withdrawBtn.title = 'Minimum balance of 0.003 XMR required';
  }
  withdrawBtn.addEventListener('click', async () => {
    withdrawBtn.disabled = true;
    withdrawBtn.textContent = 'Processing...';
    const { ok, message, error } = await initiateWithdrawal();
    if (ok) {
      alert(message || 'Withdrawal initiated successfully!');
      navigateTo('dashboard');
    } else {
      alert(error);
      withdrawBtn.disabled = false;
      withdrawBtn.textContent = 'Initiate Withdrawal';
    }
  });
  withdrawBtnDiv.appendChild(withdrawBtn);
  withdrawCard.appendChild(withdrawBtnDiv);
  container.appendChild(withdrawCard);

  // Recent transactions card
  const card = document.createElement('div');
  card.className = 'card';

  const cardHeader = document.createElement('div');
  cardHeader.className = 'card-header';

  const cardTitle = document.createElement('h3');
  cardTitle.className = 'card-title';
  cardTitle.textContent = 'Recent Transactions';

  const viewAllBtn = document.createElement('button');
  viewAllBtn.className = 'btn btn-secondary btn-sm';
  viewAllBtn.textContent = 'View All';
  viewAllBtn.addEventListener('click', () => navigateTo('transactions'));

  cardHeader.appendChild(cardTitle);
  cardHeader.appendChild(viewAllBtn);
  card.appendChild(cardHeader);

  const allTx = [...transactions.pending, ...transactions.confirmed].slice(0, 5);
  card.appendChild(createTransactionTable(allTx));
  container.appendChild(card);
}

async function renderTransactions(container, headerActions) {
  const exportBtn = document.createElement('button');
  exportBtn.className = 'btn btn-primary';
  exportBtn.textContent = 'Export CSV';
  exportBtn.addEventListener('click', exportCSV);
  headerActions.appendChild(exportBtn);

  await fetchTransactions();

  while (container.firstChild) container.removeChild(container.firstChild);

  const card = document.createElement('div');
  card.className = 'card';

  // Filters
  const filters = document.createElement('div');
  filters.className = 'filters';

  const select = document.createElement('select');
  select.className = 'filter-select';
  select.id = 'filter-status';
  [
    { value: 'all', text: 'All Status' },
    { value: 'pending', text: 'Pending' },
    { value: 'confirmed', text: 'Confirmed' },
    { value: 'transferred', text: 'Transferred' }
  ].forEach(opt => {
    const option = document.createElement('option');
    option.value = opt.value;
    option.textContent = opt.text;
    select.appendChild(option);
  });
  select.addEventListener('change', filterTransactions);
  filters.appendChild(select);
  card.appendChild(filters);

  const tableContainer = document.createElement('div');
  tableContainer.id = 'transactions-table';
  const allTx = [...transactions.pending, ...transactions.confirmed];
  tableContainer.appendChild(createTransactionTable(allTx));
  card.appendChild(tableContainer);

  container.appendChild(card);
}

function filterTransactions() {
  const status = $('filter-status').value;
  let filtered = [...transactions.pending, ...transactions.confirmed];

  if (status === 'pending') {
    filtered = transactions.pending;
  } else if (status === 'confirmed') {
    filtered = transactions.confirmed.filter(tx => tx.confirmed && !tx.transferred);
  } else if (status === 'transferred') {
    filtered = transactions.confirmed.filter(tx => tx.transferred);
  }

  const tableContainer = $('transactions-table');
  while (tableContainer.firstChild) tableContainer.removeChild(tableContainer.firstChild);
  tableContainer.appendChild(createTransactionTable(filtered));
}

async function renderPOS(container, headerActions) {
  const addBtn = document.createElement('button');
  addBtn.className = 'btn btn-primary';
  addBtn.textContent = 'Add Device';
  addBtn.addEventListener('click', () => openModal('create-pos-modal'));
  headerActions.appendChild(addBtn);

  while (container.firstChild) container.removeChild(container.firstChild);

  // Show "Connect your till" panel right after a device is created (password shown once)
  if (window.__lastPOS) {
    const d = window.__lastPOS; window.__lastPOS = null;
    const url = window.location.origin;
    const vid = (auth && auth.vendorId != null) ? auth.vendorId : 'N/A';
    const row = (k, v) => '<div class="crow"><div class="ck">' + escapeHtml(k) + '</div><div class="cv">' + escapeHtml(v) + '</div><div class="ccopy" data-v="' + escapeHtml(v) + '">&#8455; Copy</div></div>';
    const panel = document.createElement('div');
    panel.className = 'connect-till';
    panel.innerHTML =
      '<h3>📲 Connect your till</h3>' +
      '<div class="lead">Just created <strong>' + escapeHtml(d.name) + '</strong>. Enter these into the Monero Merchant Android app:</div>' +
      row('Instance URL', url) + row('Vendor ID', String(vid)) + row('Username', d.name) + row('Password', d.password) +
      '<div class="chint">🔒 The password is shown only now — copy it before leaving this page.</div>';
    container.appendChild(panel);
    panel.querySelectorAll('.ccopy').forEach(b => b.addEventListener('click', () => {
      const ok = clipCopy(b.dataset.v);
      b.textContent = ok ? 'Copied' : 'Select manually'; setTimeout(() => { b.innerHTML = '&#8455; Copy'; }, 1400);
    }));
  }

  const devices = await fetchPosDevices();

  const card = document.createElement('div');
  card.className = 'card';

  const cardHeader = document.createElement('div');
  cardHeader.className = 'card-header';
  const cardTitle = document.createElement('h3');
  cardTitle.className = 'card-title';
  cardTitle.textContent = 'Your POS Devices';
  cardHeader.appendChild(cardTitle);
  card.appendChild(cardHeader);

  if (devices.length === 0) {
    const emptyState = document.createElement('div');
    emptyState.className = 'empty-state';
    const h3 = document.createElement('h3');
    h3.textContent = 'No POS devices yet';
    const p = document.createElement('p');
    p.textContent = 'Create POS devices to accept payments. Each device gets its own login credentials.';
    emptyState.appendChild(h3);
    emptyState.appendChild(p);
    card.appendChild(emptyState);

    const btnDiv = document.createElement('div');
    btnDiv.style.textAlign = 'center';
    btnDiv.style.paddingBottom = '20px';
    const createBtn = document.createElement('button');
    createBtn.className = 'btn btn-primary';
    createBtn.textContent = 'Create Your First POS Device';
    createBtn.addEventListener('click', () => openModal('create-pos-modal'));
    btnDiv.appendChild(createBtn);
    card.appendChild(btnDiv);
  } else {
    const tableWrapper = document.createElement('div');
    tableWrapper.className = 'table-wrapper';
    const table = document.createElement('table');
    const thead = document.createElement('thead');
    const headerRow = document.createElement('tr');
    ['ID', 'Name', 'Created'].forEach(text => {
      const th = document.createElement('th');
      th.textContent = text;
      headerRow.appendChild(th);
    });
    thead.appendChild(headerRow);
    table.appendChild(thead);

    const tbody = document.createElement('tbody');
    devices.forEach(device => {
      const tr = document.createElement('tr');
      const tdId = document.createElement('td');
      tdId.textContent = '#' + device.id;
      const tdName = document.createElement('td');
      tdName.textContent = device.name;
      const tdCreated = document.createElement('td');
      tdCreated.className = 'text-muted';
      tdCreated.textContent = formatDate(device.created_at);
      tr.appendChild(tdId);
      tr.appendChild(tdName);
      tr.appendChild(tdCreated);
      tbody.appendChild(tr);
    });
    table.appendChild(tbody);
    tableWrapper.appendChild(table);
    card.appendChild(tableWrapper);
  }

  container.appendChild(card);
}

function renderSettings(container, headerActions) {
  while (container.firstChild) container.removeChild(container.firstChild);

  // Account settings card
  const accountCard = document.createElement('div');
  accountCard.className = 'card';

  const accountHeader = document.createElement('div');
  accountHeader.className = 'card-header';
  const accountTitle = document.createElement('h3');
  accountTitle.className = 'card-title';
  accountTitle.textContent = 'Account Settings';
  accountHeader.appendChild(accountTitle);
  accountCard.appendChild(accountHeader);

  const usernameGroup = document.createElement('div');
  usernameGroup.className = 'form-group';
  const usernameLabel = document.createElement('label');
  usernameLabel.className = 'form-label';
  usernameLabel.textContent = 'Username';
  const usernameInput = document.createElement('input');
  usernameInput.type = 'text';
  usernameInput.className = 'form-input';
  usernameInput.value = auth.vendorName || '';
  usernameInput.disabled = true;
  usernameGroup.appendChild(usernameLabel);
  usernameGroup.appendChild(usernameInput);
  accountCard.appendChild(usernameGroup);

  const idGroup = document.createElement('div');
  idGroup.className = 'form-group';
  const idLabel = document.createElement('label');
  idLabel.className = 'form-label';
  idLabel.textContent = 'Vendor ID';
  const idInput = document.createElement('input');
  idInput.type = 'text';
  idInput.className = 'form-input';
  idInput.value = auth.vendorId || 'N/A';
  idInput.disabled = true;
  idGroup.appendChild(idLabel);
  idGroup.appendChild(idInput);
  accountCard.appendChild(idGroup);

  container.appendChild(accountCard);

  // Change password card
  const passCard = document.createElement('div');
  passCard.className = 'card mt-20';

  const passHeader = document.createElement('div');
  passHeader.className = 'card-header';
  const passTitle = document.createElement('h3');
  passTitle.className = 'card-title';
  passTitle.textContent = 'Change Password';
  passHeader.appendChild(passTitle);
  passCard.appendChild(passHeader);

  const passForm = document.createElement('form');
  passForm.id = 'change-password-form';

  const currentPassGroup = document.createElement('div');
  currentPassGroup.className = 'form-group';
  const currentPassLabel = document.createElement('label');
  currentPassLabel.className = 'form-label';
  currentPassLabel.textContent = 'Current Password';
  const currentPassInput = document.createElement('input');
  currentPassInput.type = 'password';
  currentPassInput.id = 'current-password';
  currentPassInput.className = 'form-input';
  currentPassInput.required = true;
  currentPassGroup.appendChild(currentPassLabel);
  currentPassGroup.appendChild(currentPassInput);
  passForm.appendChild(currentPassGroup);

  const newPassGroup = document.createElement('div');
  newPassGroup.className = 'form-group';
  const newPassLabel = document.createElement('label');
  newPassLabel.className = 'form-label';
  newPassLabel.textContent = 'New Password';
  const newPassInput = document.createElement('input');
  newPassInput.type = 'password';
  newPassInput.id = 'new-password';
  newPassInput.className = 'form-input';
  newPassInput.required = true;
  newPassGroup.appendChild(newPassLabel);
  newPassGroup.appendChild(newPassInput);
  passForm.appendChild(newPassGroup);

  const passBtn = document.createElement('button');
  passBtn.type = 'submit';
  passBtn.className = 'btn btn-primary';
  passBtn.textContent = 'Update Password';
  passForm.appendChild(passBtn);

  passForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    const currentPass = $('current-password').value;
    const newPass = $('new-password').value;

    try {
      const res = await api('/auth/update-password', {
        method: 'POST',
        body: JSON.stringify({ current_password: currentPass, new_password: newPass })
      });

      if (res.ok) {
        // The old tokens are invalid after a password change
        const data = await res.json();
        auth.accessToken = data.access_token;
        auth.refreshToken = data.refresh_token;
        saveAuth();
        alert('Password updated successfully');
        $('current-password').value = '';
        $('new-password').value = '';
      } else {
        alert(await errorMessage(res, 'Failed to update password'));
      }
    } catch (e) {
      alert('Error updating password');
    }
  });

  passCard.appendChild(passForm);
  container.appendChild(passCard);

  // Danger zone
  const dangerCard = document.createElement('div');
  dangerCard.className = 'card mt-20';

  const dangerHeader = document.createElement('div');
  dangerHeader.className = 'card-header';
  const dangerTitle = document.createElement('h3');
  dangerTitle.className = 'card-title';
  dangerTitle.textContent = 'Danger Zone';
  dangerHeader.appendChild(dangerTitle);
  dangerCard.appendChild(dangerHeader);

  const dangerText = document.createElement('p');
  dangerText.className = 'text-muted';
  dangerText.textContent = 'Once you delete your account, there is no going back. Please be certain.';
  dangerCard.appendChild(dangerText);

  const deleteBtn = document.createElement('button');
  deleteBtn.className = 'btn btn-danger mt-20';
  deleteBtn.textContent = 'Delete Account';
  deleteBtn.addEventListener('click', confirmDeleteAccount);
  dangerCard.appendChild(deleteBtn);

  container.appendChild(dangerCard);
}

async function confirmDeleteAccount() {
  if (confirm('Are you sure you want to delete your account? This action cannot be undone.')) {
    const name = prompt('Type your username to confirm deletion:');
    if (name === auth.vendorName) {
      try {
        const res = await api('/vendor/delete', { method: 'POST' });
        if (res.ok) {
          alert('Account deleted');
          logout();
        } else {
          alert(await errorMessage(res, 'Failed to delete account'));
        }
      } catch (e) {
        alert('Error deleting account');
      }
    }
  }
}

// ============ Modals ============
function openModal(id) {
  $(id).classList.add('active');
}

function closeModal(id) {
  $(id).classList.remove('active');
}

// ============ Event Listeners ============
document.addEventListener('DOMContentLoaded', async () => {
  if (loadAuth() && await refreshAccessToken()) {
    showAppScreen();
  } else {
    showLoginScreen();
  }

  $('login-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const username = $('login-username').value.trim();
    const password = $('login-password').value;

    try {
      $('login-error').classList.remove('show');
      await login(username, password);
      showAppScreen();
    } catch (err) {
      $('login-error').textContent = err.message;
      $('login-error').classList.add('show');
    }
  });

  $('show-signup').addEventListener('click', (e) => {
    e.preventDefault();
    openModal('signup-modal');
  });

  $('signup-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const username = $('signup-username').value.trim();
    const email = $('signup-email').value.trim();
    const password = $('signup-password').value;
    const address = $('signup-address').value.trim();
    const invite = $('signup-invite').value.trim();

    try {
      $('signup-error').classList.remove('show');
      await signup(username, email, password, address, invite);
      closeModal('signup-modal');
      showAppScreen();
    } catch (err) {
      $('signup-error').textContent = err.message;
      $('signup-error').classList.add('show');
    }
  });

  $('create-pos-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const name = $('pos-name-input').value.trim();
    const password = $('pos-password-input').value;

    try {
      await createPOS(name, password);
      closeModal('create-pos-modal');
      window.__lastPOS = { name: name, password: password };
      $('pos-name-input').value = '';
      $('pos-password-input').value = '';
      navigateTo('pos');
    } catch (err) {
      alert(err.message);
    }
  });

  $$('.nav-item').forEach(item => {
    item.addEventListener('click', () => navigateTo(item.dataset.page));
  });

  $('logout-btn').addEventListener('click', logout);

  // Modal close buttons
  $$('[data-close]').forEach(btn => {
    btn.addEventListener('click', () => closeModal(btn.dataset.close));
  });

  // Close modals on overlay click
  $$('.modal-overlay').forEach(overlay => {
    overlay.addEventListener('click', (e) => {
      if (e.target === overlay) {
        overlay.classList.remove('active');
      }
    });
  });
});
