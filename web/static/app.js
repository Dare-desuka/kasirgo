"use strict";

const $ = sel => document.querySelector(sel);
const fmt = n => new Intl.NumberFormat("id-ID").format(n || 0);
const esc = s => String(s ?? "").replace(/[&<>"']/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));

async function api(path, opts = {}) {
  const headers = { ...(opts.body ? { "Content-Type": "application/json" } : {}), ...(opts.headers || {}) };
  if (sessionStorage.lanPin) headers["X-LAN-PIN"] = sessionStorage.lanPin;
  const res = await fetch(path, { ...opts, headers });
  const data = await res.json().catch(() => ({}));
  // ponytail: PIN hanya untuk non-localhost (server bypass loopback); desktop tak pernah kena 401.
  if (res.status === 401 && !opts._retried) {
    if (await askPin()) return api(path, { ...opts, _retried: true });
    throw new Error(data.error || "PIN salah");
  }
  if (!res.ok || data.error) throw new Error(data.error || res.statusText);
  return data;
}

let pinAsk = null;
function askPin() {
  if (pinAsk) return pinAsk;
  pinAsk = new Promise(resolve => {
    const m = modal(`
      <h3>PIN Akses HP</h3>
      <p class="muted">Lihat PIN di aplikasi desktop → Pengaturan → Akses HP.</p>
      <div class="form mt"><div><label>PIN</label><input type="password" id="pin-in" inputmode="numeric" autocomplete="off" autofocus></div></div>
      <div class="modal-actions">
        <button id="pin-cancel">Batal</button>
        <button class="primary" id="pin-ok">Masuk</button>
      </div>`);
    const done = v => { m.close(); pinAsk = null; resolve(v); };
    $("#pin-cancel").addEventListener("click", () => done(false));
    $("#pin-ok").addEventListener("click", async () => {
      const pin = $("#pin-in").value;
      const r = await fetch("/api/unlock", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ pin }) });
      if (!r.ok) { toast("PIN salah", true); return; }
      sessionStorage.lanPin = pin;
      done(true);
    });
  });
  return pinAsk;
}

function toast(msg, isErr = false) {
  const t = document.createElement("div");
  t.className = "toast" + (isErr ? " err" : "");
  t.textContent = msg;
  $("#toast-root").appendChild(t);
  setTimeout(() => t.remove(), 3000);
}

function initTheme() {
  // Default ikut preferensi OS; hanya tersimpan (localStorage) setelah user toggle manual.
  const saved = localStorage.theme;
  document.documentElement.dataset.theme = saved || (matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light");
  $("#theme-toggle").checked = document.documentElement.dataset.theme === "dark";
}
function toggleTheme() {
  const dark = document.documentElement.dataset.theme === "dark";
  document.documentElement.dataset.theme = dark ? "light" : "dark";
  localStorage.theme = document.documentElement.dataset.theme;
  $("#theme-toggle").checked = !dark;
}

function modal(html) {
  const root = $("#modal-root");
  const m = document.createElement("div");
  m.className = "modal-backdrop";
  m.innerHTML = `<div class="modal">${html}</div>`;
  root.appendChild(m);
  const close = () => m.remove();
  m.addEventListener("click", e => { if (e.target === m) close(); });
  return { close, el: m };
}

let cart = [];
let categories = [];
let settings = {};
let allProducts = [];

const UNITS = ["pcs", "kg", "gram", "liter", "botol", "dus", "bungkus"];
const METHODS = [["cash", "Cash"], ["transfer", "Transfer"], ["qris", "QRIS"], ["debit", "Debit"]];

function route() {
  const page = (location.hash.replace("#/", "") || "kasir").split("?")[0];
  document.querySelectorAll(".page").forEach(p => p.classList.remove("active"));
  document.querySelectorAll(".sidebar a").forEach(a => a.classList.toggle("active", a.dataset.page === page));
  const section = $("#page-" + page);
  if (section) section.classList.add("active");
  ({ dashboard: renderDashboard, kasir: renderKasir, produk: renderProduk, kategori: renderKategori,
     stok: renderStok, transaksi: renderTransaksi, laporan: renderLaporan, pengaturan: renderPengaturan })[page]?.();
}
window.addEventListener("hashchange", route);

async function loadBase() {
  [categories, settings] = await Promise.all([api("/api/categories"), api("/api/settings")]);
}

// ---------------- Dashboard ----------------

// donut pie dengan conic-gradient; segs = [{v, color}], center = teks di tengah.
function donut(segs, size = 140, center = "") {
  const total = segs.reduce((a, s) => a + s.v, 0) || 1;
  let acc = 0;
  const stops = segs.map(s => {
    const from = (acc / total) * 100;
    acc += s.v;
    return `${s.color} ${from.toFixed(1)}% ${((acc / total) * 100).toFixed(1)}%`;
  }).join(", ");
  return `<div class="donut" style="width:${size}px;height:${size}px;background:conic-gradient(${stops})">
    <div class="donut-hole" style="width:${Math.round(size * 0.6)}px;height:${Math.round(size * 0.6)}px">${center}</div></div>`;
}

async function renderDashboard() {
  const pg = $("#page-dashboard");
  if (!pg.classList.contains("active")) return;
  const d = await api("/api/dashboard");
  const cx = (l, fn) => `<div class="card mt"><h2>${l}</h2>` + fn() + `</div>`;
  const today = new Date().toLocaleDateString("id-ID", { weekday: "long", day: "numeric", month: "long", year: "numeric" });
  const low = d.low_stock.length, out = d.out_of_stock.length;
  const modal = Math.max(0, d.today_sales - d.today_profit);
  const pctProfit = d.today_sales ? Math.round(d.today_profit / d.today_sales * 100) : 0;
  pg.innerHTML = `
    <h1>Dashboard</h1>
    <p class="muted kicker">${today}</p>
    <div class="stats">
      <div class="stat hero"><div class="v">Rp ${fmt(d.today_profit)}</div><div class="l">Keuntungan hari ini</div></div>
      <div class="stat c-sales"><div class="v">Rp ${fmt(d.today_sales)}</div><div class="l">Penjualan hari ini</div></div>
      <div class="stat c-count"><div class="v">${d.today_count}</div><div class="l">Transaksi hari ini</div></div>
      <div class="stat c-low"><div class="v">${low}</div><div class="l">Produk hampir habis</div></div>
      <div class="stat c-out"><div class="v">${out}</div><div class="l">Produk habis</div></div>
    </div>
    <div class="dash-grid">
      ${cx("Transaksi terbaru", () => table(`
        <tr><th>Invoice</th><th class="tnum">Total</th><th>Metode</th><th>Waktu</th></tr>
        ${d.recent_tx.map(t => `<tr onclick="openTx(${t.id})" style="cursor:pointer">
          <td>${esc(t.invoice_number)}</td><td class="tnum">${fmt(t.total)}</td>
          <td>${esc(t.payment_method)}</td><td class="muted">${esc(t.created_at)}</td></tr>`).join("") || `<tr><td colspan="4" class="muted">Belum ada transaksi</td></tr>`}`))}
      ${cx("Keuntungan Hari Ini", () => `
        <div class="donut-wrap">
          ${donut([{ v: d.today_profit, color: "#16a34a" }, { v: modal, color: "#e2e8f0" }], 140, `${pctProfit}%`)}
          <div class="donut-legend">
            <div><span class="dot ok"></span>Keuntungan <b>Rp ${fmt(d.today_profit)}</b></div>
            <div><span class="dot cost"></span>Modal <b>Rp ${fmt(modal)}</b></div>
          </div>
        </div>`)}
    </div>
    <div class="dash-grid mt">
      ${cx("Produk hampir habis", () => stockTable(d.low_stock))}
      ${cx("Produk habis", () => stockTable(d.out_of_stock))}
    </div>`;
}

// table() menghindari backtick bersarang di dalam template literal induk.
const table = rows => `<table>${rows}</table>`;

const statusBadge = st => st === "out"
  ? `<span class="badge out">Habis</span>`
  : st === "low" ? `<span class="badge low">Menipis</span>` : `<span class="badge ok">Normal</span>`;

function stockTable(list) {
  if (!list.length) return `<p class="muted">Tidak ada</p>`;
  return `<table><tr><th>Produk</th><th class="tnum">Stok</th><th>Status</th></tr>` + list.map(s => `
    <tr><td>${esc(s.name)}</td><td class="tnum">${s.stock} ${esc(s.unit)}</td>
    <td>${statusBadge(s.status)}</td></tr>`).join("") + `</table>`;
}

// ---------------- Kasir ----------------

let kasirQuery = "";

async function renderKasir() {
  const pg = $("#page-kasir");
  if (!pg.classList.contains("active")) return;
  if (!allProducts.length) allProducts = await api("/api/products");
  pg.innerHTML = `
    <h1>Kasir</h1>
    <div class="cashier">
      <div>
        <div class="search-bar">
          <input id="kasir-search" placeholder="Cari produk / scan barcode / SKU … (F2)" autocomplete="off" value="${esc(kasirQuery)}">
        </div>
        <div class="product-grid" id="product-grid"></div>
      </div>
      <div class="cart-card card">
        <h2>Keranjang</h2>
        <div class="cart-items" id="cart-items"></div>
        <div class="totals">
          <div><span>Subtotal</span><span class="tnum" id="t-subtotal">Rp 0</span></div>
          <div><span>Total</span><span class="tnum grand" id="t-total">Rp 0</span></div>
          <div><span>Metode</span><select id="t-method" style="width:130px">${METHODS.map(m => `<option value="${m[0]}">${m[1]}</option>`).join("")}</select></div>
          <div><span>Bayar (F4)</span><input type="number" id="t-paid" min="0" placeholder="Masukkan nominal" style="width:130px" class="tnum"></div>
          <div><span>Kembali</span><span class="tnum" id="t-change">Rp 0</span></div>
        </div>
        <div class="pay-row">
          <button class="primary grow" id="btn-pay">BAYAR</button>
          <button id="btn-clear">Kosongkan</button>
        </div>
      </div>
    </div>`;

  const search = $("#kasir-search");
  const grid = $("#product-grid");

  function renderGrid() {
    const q = kasirQuery.trim().toLowerCase();
    const list = allProducts.filter(p => !q || p.name.toLowerCase().includes(q) || (p.barcode || "").includes(q) || (p.sku || "").includes(q));
    grid.innerHTML = list.map(p => `
      <div class="product-tile" onclick="addToCart(${p.id})">
        <div class="name">${esc(p.name)}</div>
        <div class="price">Rp ${fmt(p.selling_price)}</div>
        <div class="stock">Stok: ${p.stock} ${esc(p.unit)}</div>
      </div>`).join("") || `<p class="muted">Produk tidak ditemukan</p>`;
  }

  search.addEventListener("input", () => { kasirQuery = search.value; renderGrid(); });
  search.addEventListener("keydown", async e => {
    if (e.key !== "Enter") return;
    e.preventDefault();
    const q = search.value.trim();
    if (!q) return;
    const exact = allProducts.find(p => p.barcode === q || p.sku === q);
    if (exact) { addToCart(exact.id); search.value = ""; kasirQuery = ""; renderGrid(); return; }
    const r = await api("/api/products/search/" + encodeURIComponent(q));
    if (r.error) toast(r.error, true);
    else { addToCart(r.id); search.value = ""; kasirQuery = ""; renderGrid(); }
  });

  $("#t-paid").addEventListener("input", renderCart);
  $("#btn-pay").addEventListener("click", doCheckout);
  $("#btn-clear").addEventListener("click", () => { cart = []; renderCart(); });
  renderGrid();
  renderCart();
}

function addToCart(id) {
  const p = allProducts.find(x => x.id === id);
  if (!p) return;
  const item = cart.find(i => i.id === id);
  if (item) item.qty++;
  else cart.push({ ...p, qty: 1 });
  renderCart();
  // Setelah produk masuk keranjang, fokus lompat ke kolom bayar — kecuali sedang scan
  // (fokus masih di kotak cari), supaya scan barang berikutnya tidak ketik di bayar.
  const search = document.getElementById("kasir-search");
  if (document.activeElement !== search) document.getElementById("t-paid")?.focus();
}

function renderCart() {
  const wrap = $("#cart-items"); if (!wrap) return;
  wrap.innerHTML = cart.map((i, idx) => `
    <div class="cart-item">
      <div>${esc(i.name)}<div class="muted">Rp ${fmt(i.selling_price)}</div></div>
      <div class="qty">
        <button onclick="changeQty(${idx},-1)">−</button>
        <input type="number" min="1" value="${i.qty}" onchange="setQty(${idx},this.value)">
        <button onclick="changeQty(${idx},1)">+</button>
      </div>
      <div class="tnum">${fmt(i.selling_price * i.qty)}</div>
      <button class="danger" onclick="removeItem(${idx})">✕</button>
    </div>`).join("") || `<p class="muted">Keranjang kosong</p>`;

  const subtotal = cart.reduce((s, i) => s + i.selling_price * i.qty, 0);
  const paid = parseInt($("#t-paid").value || "0", 10);
  $("#t-subtotal").textContent = "Rp " + fmt(subtotal);
  $("#t-total").textContent = "Rp " + fmt(subtotal);
  $("#t-change").textContent = "Rp " + fmt(Math.max(0, paid - subtotal));
}

function changeQty(idx, d) { cart[idx].qty = Math.max(1, cart[idx].qty + d); renderCart(); }
function setQty(idx, v) { cart[idx].qty = Math.max(1, parseInt(v || "1", 10)); renderCart(); }
function removeItem(idx) { cart.splice(idx, 1); renderCart(); }

async function doCheckout() {
  if (!cart.length) return toast("Keranjang kosong", true);
  const body = {
    items: cart.map(i => ({ product_id: i.id, quantity: i.qty })),
    paid: parseInt($("#t-paid").value || "0", 10),
    payment_method: $("#t-method").value,
    cashier: "",
  };
  try {
    const tx = await api("/api/transactions", { method: "POST", body: JSON.stringify(body) });
    cart = [];
    renderCart();
    showReceipt(tx);
    allProducts = await api("/api/products");
  } catch (e) { toast(e.message, true); }
}

function showReceipt(tx) {
  const m = modal(`
    <h3>Struk ${esc(tx.invoice_number)}</h3>
    <div class="receipt">${receiptHTML(tx)}</div>
    <div class="modal-actions">
      <button onclick="print()">Print</button>
      <button class="primary" onclick="document.querySelector('.modal-backdrop').remove()">Tutup</button>
    </div>`);
  m.el.querySelector("button").addEventListener("click", print);
}

function receiptHTML(tx) {
  const s = settings;
  const lines = [
    s.store_name || "TOKO",
    s.store_address || "",
    s.store_phone || "",
    "------------------------------",
  ];
  tx.items.forEach(i => {
    lines.push(`${i.product_name}`);
    lines.push(`${i.quantity} x ${fmt(i.price)}      ${fmt(i.subtotal)}`);
  });
  lines.push("------------------------------");
  lines.push(`Subtotal      ${fmt(tx.subtotal)}`);
  lines.push(`TOTAL         ${fmt(tx.total)}`);
  lines.push(`Bayar         ${fmt(tx.paid)}`);
  lines.push(`Kembali       ${fmt(tx.change)}`);
  lines.push("------------------------------");
  lines.push(s.receipt_footer || "Terima kasih");
  return `<div class="center">${lines.map(l => esc(l)).join("<br>")}</div>`;
}

// ---------------- Produk ----------------

async function renderProduk() {
  const pg = $("#page-produk");
  if (!pg.classList.contains("active")) return;
  const q = location.hash.split("?")[1] || "";
  pg.innerHTML = `
    <h1>Produk</h1>
    <div class="toolbar">
      <input class="grow" id="prod-q" placeholder="Cari nama / barcode / SKU" value="${esc(q)}">
      <select id="prod-cat" style="width:180px"><option value="0">Semua kategori</option></select>
      <button class="primary" onclick="openProductForm()">+ Produk</button>
    </div>
    <div class="card"><div id="prod-table"></div></div>`;

  const catSel = $("#prod-cat");
  catSel.innerHTML += categories.map(c => `<option value="${c.id}">${esc(c.name)}</option>`).join("");

  async function load() {
    const qq = encodeURIComponent($("#prod-q").value);
    const cc = $("#prod-cat").value;
    const list = await api(`/api/products?q=${qq}&category=${cc}`);
    $("#prod-table").innerHTML = `
      <table><tr><th>Nama</th><th>Barcode</th><th>SKU</th><th>Kategori</th><th class="tnum">Harga Beli</th><th class="tnum">Harga Jual</th><th class="tnum">Stok</th><th>Satuan</th><th></th></tr>
      ${list.map(p => `<tr>
        <td>${esc(p.name)}</td><td class="muted">${esc(p.barcode || "-")}</td><td class="muted">${esc(p.sku || "-")}</td>
        <td>${esc(p.category_name || "-")}</td><td class="tnum">${fmt(p.purchase_price)}</td><td class="tnum">${fmt(p.selling_price)}</td>
        <td class="tnum">${p.stock} ${esc(p.unit)}</td><td>${esc(p.unit)}</td>
        <td class="nowrap"><button onclick="openProductForm(${p.id})">Edit</button>
        <button class="danger" onclick="deleteProduct(${p.id})">Hapus</button></td></tr>`).join("") ||
        `<tr><td colspan="9" class="muted">Tidak ada produk</td></tr>`}</table>`;
  }
  $("#prod-q").addEventListener("input", load);
  $("#prod-q").addEventListener("keydown", async e => {
    if (e.key !== "Enter") return;
    const q = e.target.value.trim();
    if (!q) return;
    const r = await api("/api/products/search/" + encodeURIComponent(q)).catch(e => ({ error: e.message }));
    if (!r.error) return;
    // Barcode/SKU (tanpa spasi) yang tidak ditemukan → langsung buka form tambah produk, barcode terisi.
    if (!/\s/.test(q)) {
      openProductForm(0, { barcode: q });
      toast(`Barcode "${q}" tidak ada — tambahkan produk baru`);
    } else {
      toast("Produk tidak ditemukan", true);
    }
  });
  catSel.addEventListener("change", load);
  load();
}

function openProductForm(id, prefill = {}) {
  const p = id ? allProducts.find(x => x.id === id) : { category_id: categories[0]?.id || null, ...prefill };
  const m = modal(`
    <h3>${id ? "Edit Produk" : "Produk Baru"}</h3>
    <div class="form">
      <div><label>Nama *</label><input id="f-name" value="${esc(p.name || "")}" required></div>
      <div class="row">
        <div><label>Barcode</label><input id="f-barcode" value="${esc(p.barcode || "")}"></div>
        <div><label>SKU</label><input id="f-sku" value="${esc(p.sku || "")}"></div>
      </div>
      <div class="row">
        <div><label>Kategori</label><select id="f-cat">${categories.map(c => `<option value="${c.id}" ${p.category_id === c.id ? "selected" : ""}>${esc(c.name)}</option>`).join("")}</select></div>
        <div><label>Satuan</label><select id="f-unit">${UNITS.map(u => `<option ${(p.unit || "pcs") === u ? "selected" : ""}>${u}</option>`).join("")}</select></div>
      </div>
      <div class="row">
        <div><label>Harga Beli (Rp)</label><input type="number" id="f-buy" value="${p.purchase_price || 0}" min="0"></div>
        <div><label>Harga Jual (Rp)</label><input type="number" id="f-sell" value="${p.selling_price || 0}" min="0"></div>
      </div>
      <div class="row">
        <div><label>Stok</label><input type="number" id="f-stock" value="${p.stock || 0}" min="0"></div>
        <div><label>Stok Minimum</label><input type="number" id="f-min" value="${p.minimum_stock || 0}" min="0"></div>
      </div>
    </div>
    <div class="modal-actions">
      <button onclick="document.querySelector('.modal-backdrop').remove()">Batal</button>
      <button class="primary" id="f-save">Simpan</button>
    </div>`);

  $("#f-save").addEventListener("click", async () => {
    const body = {
      id: id || 0,
      name: $("#f-name").value,
      barcode: $("#f-barcode").value,
      sku: $("#f-sku").value,
      category_id: $("#f-cat").value ? parseInt($("#f-cat").value) : null,
      unit: $("#f-unit").value,
      purchase_price: parseInt($("#f-buy").value || "0", 10),
      selling_price: parseInt($("#f-sell").value || "0", 10),
      stock: parseInt($("#f-stock").value || "0", 10),
      minimum_stock: parseInt($("#f-min").value || "0", 10),
    };
    if (!body.name) return toast("Nama wajib diisi", true);
    try {
      if (id) {
        const prev = allProducts.find(x => x.id === id);
        if (prev && prev.stock !== body.stock) {
          await api("/api/stock/adjust", { method: "POST", body: JSON.stringify({ product_id: id, delta: body.stock - prev.stock, note: "edit produk" }) });
        }
        await api(`/api/products/${id}`, { method: "PUT", body: JSON.stringify(body) });
      } else {
        await api("/api/products", { method: "POST", body: JSON.stringify(body) });
      }
      toast("Produk disimpan");
      m.close();
      allProducts = await api("/api/products");
      renderProduk();
    } catch (e) { toast(e.message, true); }
  });
}

async function deleteProduct(id) {
  if (!confirm("Hapus produk ini?")) return;
  try {
    await api(`/api/products/${id}`, { method: "DELETE" });
    toast("Produk dihapus");
    allProducts = await api("/api/products");
    renderProduk();
  } catch (e) { toast(e.message, true); }
}

// ---------------- Kategori ----------------

async function renderKategori() {
  const pg = $("#page-kategori");
  if (!pg.classList.contains("active")) return;
  pg.innerHTML = `
    <h1>Kategori</h1>
    <div class="toolbar">
      <input id="k-name" placeholder="Nama kategori baru" style="max-width:280px">
      <button class="primary" onclick="addCategory()">Tambah</button>
    </div>
    <div class="card" id="k-list"></div>`;
  $("#k-name").addEventListener("keydown", e => { if (e.key === "Enter") addCategory(); });
  const list = await api("/api/categories");
  $("#k-list").innerHTML = `
    <table><tr><th>Nama</th><th></th></tr>` +
    list.map(c => `<tr><td>${esc(c.name)}</td>
      <td class="nowrap"><button onclick="renameCategory(${c.id})">Ubah</button>
      <button class="danger" onclick="deleteCategory(${c.id})">Hapus</button></td></tr>`).join("") +
    `</table>`;
}

async function addCategory() {
  const name = $("#k-name").value.trim();
  if (!name) return toast("Nama wajib diisi", true);
  try {
    await api("/api/categories", { method: "POST", body: JSON.stringify({ name }) });
    categories = await api("/api/categories");
    renderKategori();
  } catch (e) { toast(e.message, true); }
}

async function renameCategory(id) {
  const name = prompt("Nama kategori baru:", categories.find(c => c.id === id)?.name);
  if (!name) return;
  try { await api(`/api/categories/${id}`, { method: "PUT", body: JSON.stringify({ name: name.trim() }) }); categories = await api("/api/categories"); renderKategori(); }
  catch (e) { toast(e.message, true); }
}

async function deleteCategory(id) {
  if (!confirm("Hapus kategori ini?")) return;
  try { await api(`/api/categories/${id}`, { method: "DELETE" }); categories = await api("/api/categories"); renderKategori(); }
  catch (e) { toast(e.message, true); }
}

// ---------------- Stok ----------------

async function renderStok() {
  const pg = $("#page-stok");
  if (!pg.classList.contains("active")) return;
  const list = await api("/api/reports/stock");
  pg.innerHTML = `
    <h1>Stok</h1>
    <div class="toolbar">
      <input class="grow" id="stok-scan" placeholder="Scan barcode produk → buka penyesuaian stok" autocomplete="off">
      <button id="stok-cam" style="min-height:44px">📷 Scan</button>
    </div>
    <div class="card">
      <table><tr><th>Produk</th><th>Barcode</th><th class="tnum">Stok</th><th class="tnum">Min</th><th>Satuan</th><th>Status</th><th></th></tr>
      ${list.map(s => `<tr>
        <td>${esc(s.name)}</td><td class="muted">${esc(s.barcode || "-")}</td>
        <td class="tnum">${s.stock} ${esc(s.unit)}</td><td class="tnum">${s.minimum_stock}</td><td>${esc(s.unit)}</td>
        <td>${statusBadge(s.status)}</td>
        <td><button onclick="openAdjust(${s.product_id}, ${s.stock}, '${esc(s.name).replace(/'/g, "\\'")}')">Penyesuaian</button></td></tr>`).join("")}
      </table>
    </div>
    <h2 class="mt">Riwayat Stok</h2>
    <div class="card"><div id="mov-table"></div></div>`;

  $("#stok-scan").addEventListener("keydown", async e => {
    if (e.key !== "Enter") return;
    const q = e.target.value.trim();
    if (!q) return;
    try {
      const r = await api("/api/products/search/" + encodeURIComponent(q));
      openAdjust(r.id, r.stock, r.name); e.target.value = "";
    } catch (err) { toast(err.message, true); }
  });
  $("#stok-cam").addEventListener("click", openCamScan);
  const ms = await api("/api/stock/movements");
  $("#mov-table").innerHTML = `
    <table><tr><th>Waktu</th><th>Produk</th><th>Tipe</th><th class="tnum">Jumlah</th><th>Catatan</th></tr>` +
    ms.map(m => `<tr><td class="muted nowrap">${esc(m.created_at)}</td><td>${esc(m.product_name || "-")}</td>
      <td>${esc(m.type)}</td><td class="tnum">${m.quantity > 0 ? "+" : ""}${m.quantity}</td><td class="muted">${esc(m.note || "")}</td></tr>`).join("") +
    `</table>`;
}

function openAdjust(id, current, name) {
  const m = modal(`
    <h3>Penyesuaian Stok</h3>
    <p class="muted">${esc(name)} — stok sekarang <b>${current}</b></p>
    <div class="form mt">
      <div><label>Penyesuaian (+/-)</label><input type="number" id="a-delta" inputmode="numeric" placeholder="-2 atau +10" autofocus></div>
      <div><label>Alasan</label><input id="a-note" placeholder="Barang rusak, stok masuk …"></div>
    </div>
    <div class="modal-actions">
      <button onclick="document.querySelector('.modal-backdrop').remove()">Batal</button>
      <button class="primary" id="a-save">Simpan</button>
    </div>`);
  $("#a-save").addEventListener("click", async () => {
    const delta = parseInt($("#a-delta").value || "0", 10);
    if (!delta) return toast("Penyesuaian tidak boleh 0", true);
    try {
      await api("/api/stock/adjust", { method: "POST", body: JSON.stringify({ product_id: id, delta, note: $("#a-note").value }) });
      toast("Stok disesuaikan");
      m.close();
      allProducts = await api("/api/products");
      renderStok();
    } catch (e) { toast(e.message, true); }
  });
}

// Scan barcode browser: native BarcodeDetector saja (tanpa library).
// Untuk scan serius pakai aplikasi HP KasirGo Stok (kamera native, tanpa flag).
// ponytail: tanpa dependency, tanpa bundel; browser yang tak dukung diarahkan ke APK.
async function openCamScan() {
  const m = modal(`
    <h3>Scan Barcode</h3>
    <p class="muted" id="cam-status">Mengaktifkan kamera…</p>
    <video id="cam-video" class="cam-video" playsinline muted></video>
    <div class="form mt">
      <div><label>atau foto barcode</label><input type="file" id="cam-file" accept="image/*" capture="environment"></div>
    </div>
    <div class="modal-actions"><button id="cam-close">Tutup</button></div>`);
  const video = $("#cam-video"), status = $("#cam-status");
  const formats = ["ean_13", "ean_8", "upc_a", "code_128", "qr_code"];
  if (!("BarcodeDetector" in window)) {
    status.textContent = "Browser ini tak mendukung scan — pakai aplikasi HP KasirGo Stok.";
    return;
  }
  let stream = null, timer = null;
  const stop = () => { clearInterval(timer); stream?.getTracks().forEach(t => t.stop()); };
  $("#cam-close").addEventListener("click", () => { stop(); m.close(); });
  const found = async code => {
    stop(); m.close();
    try {
      const r = await api("/api/products/search/" + encodeURIComponent(code));
      openAdjust(r.id, r.stock, r.name);
    } catch (e) { toast(e.message, true); }
  };
  $("#cam-file").addEventListener("change", async e => {
    const f = e.target.files[0];
    if (!f) return;
    try {
      const rs = await new BarcodeDetector({ formats }).detect(await createImageBitmap(f));
      if (rs[0]?.rawValue) found(rs[0].rawValue);
      else toast("Barcode tak terbaca dari foto", true);
    } catch (err) { toast(err.message || "Barcode tak terbaca", true); }
  });
  try {
    stream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: "environment" } });
    video.srcObject = stream;
    await video.play();
    status.textContent = "Arahkan kamera ke barcode…";
    const det = new BarcodeDetector({ formats });
    timer = setInterval(async () => {
      try {
        const rs = await det.detect(video);
        if (rs[0]?.rawValue) found(rs[0].rawValue);
      } catch (_) { /* abaikan frame gagal */ }
    }, 500);
  } catch (_) {
    status.textContent = "Kamera tak bisa dibuka — pakai foto di atas atau aplikasi HP KasirGo Stok.";
  }
}

// ---------------- Transaksi ----------------

async function renderTransaksi() {
  const pg = $("#page-transaksi");
  if (!pg.classList.contains("active")) return;
  const today = new Date().toISOString().slice(0, 10);
  pg.innerHTML = `
    <h1>Transaksi</h1>
    <div class="toolbar">
      <input type="date" id="tx-from" style="width:auto">
      <input type="date" id="tx-to" style="width:auto">
      <button class="primary" onclick="loadTx()">Cari</button>
    </div>
    <div class="card" id="tx-table"></div>`;
  const d1 = new Date(); d1.setDate(d1.getDate() - 6);
  $("#tx-from").value = d1.toISOString().slice(0, 10);
  $("#tx-to").value = today;
  loadTx();
}

async function loadTx() {
  const from = $("#tx-from").value, to = $("#tx-to").value;
  const list = await api(`/api/transactions?from=${from}&to=${to}`);
  $("#tx-table").innerHTML = `
    <table><tr><th>Invoice</th><th class="tnum">Total</th><th>Bayar</th><th>Metode</th><th>Waktu</th></tr>` +
    list.map(t => `<tr onclick="openTx(${t.id})" style="cursor:pointer">
      <td>${esc(t.invoice_number)}</td><td class="tnum">${fmt(t.total)}</td>
      <td class="tnum">${fmt(t.paid)}</td><td>${esc(t.payment_method)}</td><td class="muted">${esc(t.created_at)}</td></tr>`).join("") +
    `</table>`;
}

async function openTx(id) {
  const t = await api("/api/transactions/" + id);
  modal(`
    <h3>${esc(t.invoice_number)}</h3>
    <div class="receipt">${receiptHTML(t)}</div>
    <div class="modal-actions">
      <button onclick="print()">Print</button>
      <button class="primary" onclick="document.querySelector('.modal-backdrop').remove()">Tutup</button>
    </div>`);
}

// ---------------- Laporan ----------------

async function renderLaporan() {
  const pg = $("#page-laporan");
  if (!pg.classList.contains("active")) return;
  const today = new Date().toISOString().slice(0, 10);
  const presets = [
    ["today", "Hari ini"], ["yesterday", "Kemarin"], ["week", "Minggu ini"], ["month", "Bulan ini"], ["custom", "Custom"],
  ];
  pg.innerHTML = `
    <h1>Laporan</h1>
    <div class="toolbar">
      <div class="seg" id="rp-seg">${presets.map(([k, l]) => `<button data-rp="${k}">${l}</button>`).join("")}</div>
      <input type="date" id="rp-from" style="width:auto">
      <input type="date" id="rp-to" style="width:auto">
    </div>
    <div class="card" id="rp-sales"></div>
    <h2 class="mt">Produk Terlaris</h2>
    <div class="card" id="rp-top"></div>
    <h2 class="mt">Laporan Stok</h2>
    <div class="card" id="rp-stock"></div>`;

  $("#rp-seg").querySelectorAll("button").forEach(b => b.addEventListener("click", async () => {
    $("#rp-seg").querySelectorAll("button").forEach(x => x.classList.remove("on"));
    b.classList.add("on");
    const k = b.dataset.rp;
    let [from, to] = [today, today];
    if (k === "yesterday") { const d = new Date(Date.now() - 864e5); from = to = d.toISOString().slice(0, 10); }
    if (k === "week") { const d = new Date(); d.setDate(d.getDate() - 6); from = d.toISOString().slice(0, 10); }
    if (k === "month") { const d = new Date(); d.setDate(1); from = d.toISOString().slice(0, 10); }
    if (k === "custom") {
      from = $("#rp-from").value || today; to = $("#rp-to").value || today;
      if (from > to) [from, to] = [to, from];
    }
    await loadSales(from, to);
  }));
  $("#rp-from").addEventListener("change", () => $("#rp-seg button[data-rp=custom]").click());
  $("#rp-to").addEventListener("change", () => $("#rp-seg button[data-rp=custom]").click());
  $("#rp-seg button[data-rp=today]").click();

  const stock = await api("/api/reports/stock");
  $("#rp-stock").innerHTML = stockTable(stock);
}

async function loadSales(from, to) {
  const r = await api(`/api/reports/sales?from=${from}&to=${to}`);
  $("#rp-sales").innerHTML = `
    <div class="stats" style="margin-bottom:0">
      <div class="stat"><div class="v">${r.count}</div><div class="l">Transaksi</div></div>
      <div class="stat"><div class="v">Rp ${fmt(r.gross)}</div><div class="l">Penjualan</div></div>
      <div class="stat"><div class="v">Rp ${fmt(r.net)}</div><div class="l">Net</div></div>
    </div>`;
  $("#rp-top").innerHTML = `
    <table><tr><th>Produk</th><th class="tnum">Terjual</th><th class="tnum">Total</th></tr>` +
    r.top_products.map(p => `<tr><td>${esc(p.product_name)}</td><td class="tnum">${p.quantity}</td><td class="tnum">${fmt(p.revenue)}</td></tr>`).join("") ||
    `<p class="muted">Tidak ada data</p></table>`;
}

// ---------------- Pengaturan ----------------

async function renderPengaturan() {
  const pg = $("#page-pengaturan");
  if (!pg.classList.contains("active")) return;
  const s = settings;
  pg.innerHTML = `
    <h1>Pengaturan</h1>
    <div class="settings-grid">
      <div class="card">
        <h2>Informasi Toko</h2>
        <div class="form">
          <div><label>Nama Toko</label><input id="s-name" value="${esc(s.store_name || "")}"></div>
          <div><label>Alamat</label><input id="s-addr" value="${esc(s.store_address || "")}"></div>
          <div><label>Telepon</label><input id="s-phone" value="${esc(s.store_phone || "")}"></div>
          <div><label>Mata Uang</label><select id="s-currency"><option ${s.currency !== "IDR" ? "" : "selected"}>IDR</option></select></div>
          <div><label>Footer Struk</label><input id="s-footer" value="${esc(s.receipt_footer || "")}"></div>
        </div>
        <button class="primary mt" onclick="saveSettings()">Simpan</button>
      </div>
      <div class="card">
        <h2>Akses HP (WiFi sama)</h2>
        <p class="muted">Buka alamat ini di browser HP. Alamat selalu ikut IP laptop sekarang — kalau IP berubah, refresh halaman ini.</p>
        <div id="lan-urls" class="mt"><p class="muted">Memuat alamat…</p></div>
        <div class="mt"><label>PIN Akses HP (kosong = bebas)</label><input type="text" id="s-pin" inputmode="numeric" value="${esc(s.lan_pin || "")}" autocomplete="off" placeholder="mis. 1234"></div>
        <p class="muted" id="pin-status"></p>
        <button class="primary mt" onclick="savePin()">Simpan PIN</button>
      </div>
      <div class="card">
        <h2>Data Transaksi</h2>
        <p class="muted">Hapus data transaksi (beserta item & gerak stoknya). Stok produk tidak dikembalikan.</p>
        <div class="row mt">
          <div><label>Dari</label><input type="date" id="del-from"></div>
          <div><label>Sampai</label><input type="date" id="del-to"></div>
          <button class="danger" style="align-self:end" onclick="deleteTransactions(false)">Hapus Rentang</button>
        </div>
        <button class="danger mt" onclick="deleteTransactions(true)">Hapus Semua Transaksi</button>
      </div>
      <div class="card full">
        <h2>Database</h2>
        <p class="muted">Lokasi database otomatis di direktori data aplikasi — tidak bergantung folder tempat program dijalankan.</p>
        <div class="row mt">
          <button class="primary" onclick="backupDB()">Backup Database</button>
          <label class="row" style="align-items:center"><button onclick="document.getElementById('restore-file').click();return false">Restore Database</button>
          <input type="file" id="restore-file" accept=".db,.sqlite" style="display:none"></label>
        </div>
      </div>
    </div>`;
  refreshPinStatus();
  api("/api/network").then(n => {
    const el = $("#lan-urls");
    if (!el) return;
    el.innerHTML = n.urls?.length
      ? n.urls.map(u => `<div class="big-url">${esc(u)}</div>`).join("")
      : `<p class="muted">Tidak ada IP LAN — sambungkan laptop ke WiFi dulu.</p>`;
  }).catch(() => {
    const el = $("#lan-urls");
    if (el) el.innerHTML = `<p class="muted">Gagal memuat alamat.</p>`;
  });
}

// ponytail: PIN punya tombol sendiri agar tak dikira ikut Simpan toko; status dibaca dari settings.
function refreshPinStatus() {
  const el = $("#pin-status");
  if (!el) return;
  el.textContent = settings.lan_pin
    ? `PIN tersimpan: ${settings.lan_pin} — HP wajib isi PIN ini.`
    : "Tanpa PIN — semua HP satu WiFi bisa langsung masuk.";
}

async function savePin() {
  try {
    await api("/api/settings", { method: "PUT", body: JSON.stringify({ lan_pin: $("#s-pin").value }) });
    settings = { ...settings, lan_pin: $("#s-pin").value };
    refreshPinStatus();
    toast("PIN disimpan — berlaku langsung, tanpa restart");
  } catch (e) { toast(e.message, true); }
}

async function saveSettings() {
  const body = {
    store_name: $("#s-name").value, store_address: $("#s-addr").value,
    store_phone: $("#s-phone").value, currency: $("#s-currency").value,
    receipt_footer: $("#s-footer").value, lan_pin: $("#s-pin")?.value ?? settings.lan_pin ?? "",
  };
  try { await api("/api/settings", { method: "PUT", body: JSON.stringify(body) }); settings = body; toast("Pengaturan disimpan"); }
  catch (e) { toast(e.message, true); }
}

function backupDB() {
  window.location = "/api/backup/download";
  toast("Menyiapkan backup…");
}

async function deleteTransactions(all) {
  const from = all ? "" : $("#del-from").value;
  const to = all ? "" : $("#del-to").value;
  const what = all ? "SEMUA transaksi" : `transaksi ${from || "awal"} s/d ${to || "sekarang"}`;
  if (!all && !from && !to) { toast("Isi minimal satu tanggal dulu", true); return; }
  if (!confirm(`Hapus ${what}? Tindakan ini permanen dan tidak bisa dibatalkan.`)) return;
  try {
    const r = await api(`/api/transactions?from=${from}&to=${to}`, { method: "DELETE" });
    toast(`${r.deleted} transaksi dihapus`);
  } catch (e) { toast(e.message, true); }
}

document.addEventListener("change", async e => {
  if (e.target.id !== "restore-file") return;
  const f = e.target.files[0];
  if (!f) return;
  if (!confirm(`Restore database dari "${f.name}"? Data saat ini akan diganti.`)) { e.target.value = ""; return; }
  const fd = new FormData();
  fd.append("backup", f);
  try {
    const r = await api("/api/restore", { method: "POST", body: fd });
    toast(r.message);
    setTimeout(() => location.reload(), 800);
  } catch (err) { toast(err.message, true); }
  e.target.value = "";
});

// ---------------- Keyboard ----------------

document.addEventListener("keydown", e => {
  if (e.key === "Escape" && $("#modal-root").children.length) {
    $("#modal-root").lastChild.remove();
    return;
  }
  if (e.key === "F2") { e.preventDefault(); location.hash = "#/kasir"; setTimeout(() => $("#kasir-search")?.focus(), 50); }
  if (e.key === "F4") { e.preventDefault(); location.hash = "#/kasir"; setTimeout(() => $("#t-paid")?.focus(), 80); }
  const inModal = $("#modal-root").children.length;
  if (e.key === "Enter" && inModal) { const save = $(".modal .primary"); if (save) save.click(); }
});

// ---------------- Input helpers ----------------
// Auto kapital huruf pertama untuk input nama
// Hapus leading zero untuk input angka
document.addEventListener("input", e => {
  const t = e.target;
  if (t.tagName !== "INPUT") return;
  if (t.id === "f-name" || t.id === "k-name" || t.id === "s-name") {
    const v = t.value;
    if (v && v[0] >= "a" && v[0] <= "z") t.value = v[0].toUpperCase() + v.slice(1);
  }
  if (t.type === "number" && t.value.length > 1 && t.value[0] === "0" && t.value[1] !== ".") {
    t.value = t.value.replace(/^0+(?=\d)/, "");
  }
});

// ---------------- Init ----------------

(async function init() {
  initTheme();
  $("#theme-toggle").addEventListener("change", toggleTheme);
  await loadBase();
  if (!location.hash) location.hash = "#/kasir";
  route();
})();