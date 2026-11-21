// Highlight active navigation link and manage collapsible dropdown state
function updateActiveNavLink() {
    const currentPath = window.location.pathname;

    // Highlight main links (excluding dropdown links)
    document.querySelectorAll(".sidebar-nav > a").forEach(link => {
        const href = link.getAttribute("href");
        const isActive = href === "/" ? currentPath === "/" : (currentPath.startsWith(href) || (href === "/calendar" && currentPath.startsWith("/reminders")));
        if (isActive) {
            link.classList.add("active");
        } else {
            link.classList.remove("active");
        }
    });

    // Highlight dropdown links dynamically and auto-expand active dropdowns
    document.querySelectorAll(".sidebar-dropdown").forEach(dropdown => {
        let hasActive = false;
        const links = dropdown.querySelectorAll(".sidebar-dropdown-menu a");
        // First pass: check for exact matches or child-page matches (/items/new -> /items)
        links.forEach(link => {
            const href = link.getAttribute("href");
            if (currentPath === href || currentPath.startsWith(href + "/")) {
                link.classList.add("active");
                hasActive = true;
            } else {
                link.classList.remove("active");
            }
        });
        // Second pass: if multiple links are active (e.g. /inventory + /inventory/monthly),
        // keep only the most specific (longest href) match
        const activeLinks = dropdown.querySelectorAll(".sidebar-dropdown-menu a.active");
        if (activeLinks.length > 1) {
            let longest = activeLinks[0];
            activeLinks.forEach(link => {
                if ((link.getAttribute("href") || "").length > (longest.getAttribute("href") || "").length) {
                    longest = link;
                }
            });
            activeLinks.forEach(link => {
                if (link !== longest) link.classList.remove("active");
            });
            hasActive = true;
        }
        if (hasActive) {
            dropdown.classList.add("open");
        } else {
            dropdown.classList.remove("open");
        }
    });

    // Highlight profile menu links
    document.querySelectorAll(".profile-menu-item").forEach(link => {
        const href = link.getAttribute("href");
        if (href && (currentPath === href || currentPath.startsWith(href + "/"))) {
            link.classList.add("active");
        } else {
            link.classList.remove("active");
        }
    });

    // Highlight profile button if on an admin/profile sub-page
    const profileBtn = document.getElementById("profile-chip-btn");
    if (profileBtn) {
        const isAdminPath = currentPath.startsWith("/settings/users") ||
            currentPath.startsWith("/settings/roles") ||
            currentPath.startsWith("/settings/system") ||
            currentPath.startsWith("/activity-logs");
        if (isAdminPath) {
            profileBtn.classList.add("active");
        } else {
            profileBtn.classList.remove("active");
        }
    }

    if (typeof updateDynamicQuickMenu === "function") {
        updateDynamicQuickMenu();
    }
}

// Toggle Profile Popover Sub-Menu
function toggleProfileMenu(event) {
    if (event) {
        event.preventDefault();
        event.stopPropagation();
    }
    const menu = document.getElementById("profile-menu");
    const btn = document.getElementById("profile-chip-btn");
    if (!menu) return;
    const isOpen = menu.classList.toggle("show");
    if (btn) {
        btn.setAttribute("aria-expanded", isOpen ? "true" : "false");
        const caret = btn.querySelector(".profile-chip-caret");
        if (caret) caret.style.transform = isOpen ? "rotate(180deg)" : "rotate(0deg)";
    }
}

function closeProfileMenu() {
    const menu = document.getElementById("profile-menu");
    const btn = document.getElementById("profile-chip-btn");
    if (menu) menu.classList.remove("show");
    if (btn) {
        btn.setAttribute("aria-expanded", "false");
        const caret = btn.querySelector(".profile-chip-caret");
        if (caret) caret.style.transform = "rotate(0deg)";
    }
}

// Outside click and ESC handler for Quick Create Menu & Profile Menu
document.addEventListener("click", function (evt) {
    const menu = document.getElementById("quick-create-menu");
    const btn = document.getElementById("quick-create-btn");
    const mobileBtn = evt.target.closest(".quick-create-btn-mobile");
    if (menu && menu.classList.contains("show")) {
        if (!menu.contains(evt.target) && !btn?.contains(evt.target) && !mobileBtn) {
            closeQuickCreateMenu();
        }
    }

    const profMenu = document.getElementById("profile-menu");
    const profBtn = document.getElementById("profile-chip-btn");
    if (profMenu && profMenu.classList.contains("show")) {
        if (!profMenu.contains(evt.target) && !profBtn?.contains(evt.target)) {
            closeProfileMenu();
        }
    }
});

document.addEventListener("keydown", function (evt) {
    if (evt.key === "Escape") {
        closeQuickCreateMenu();
        closeProfileMenu();
    }
});

// Toggle sidebar dropdown on desktop/mobile
function toggleSidebarDropdown(event) {
    event.preventDefault();
    // In collapsed sidebar state on desktop, dropdown menus should only be hovered, not clicked
    const sidebar = document.getElementById("sidebar");
    if (sidebar && sidebar.classList.contains("collapsed") && window.innerWidth > 768) {
        return;
    }
    const dropdown = event.currentTarget.closest(".sidebar-dropdown");
    if (dropdown) {
        dropdown.classList.toggle("open");
    }
}

// Toggle desktop sidebar collapse state
function toggleSidebarCollapse() {
    const sidebar = document.getElementById("sidebar");
    if (sidebar) {
        const isCollapsed = sidebar.classList.toggle("collapsed");
        localStorage.setItem("sidebar-collapsed", isCollapsed ? "true" : "false");
    }
}

// Toggle mobile sidebar visibility
function toggleMobileSidebar() {
    const sidebar = document.getElementById("sidebar");
    const backdrop = document.getElementById("sidebar-backdrop");
    if (sidebar && backdrop) {
        sidebar.classList.toggle("open");
        backdrop.classList.toggle("show");
    }
}

// Close mobile sidebar on navigation click
function closeMobileSidebarOnNav() {
    const sidebar = document.getElementById("sidebar");
    const backdrop = document.getElementById("sidebar-backdrop");
    if (sidebar && backdrop) {
        sidebar.classList.remove("open");
        backdrop.classList.remove("show");
    }
}

// Quick Actions dynamic handler
function handleQuickAction(action, url, event) {
    if (event) event.preventDefault();
    closeQuickCreateMenu();

    if (action === 'receive-stock') {
        if (typeof window.toggleReceiveStockForm === 'function' && document.getElementById('receive-stock-card')) {
            window.toggleReceiveStockForm();
            return;
        }
        htmx.ajax('GET', url || '/inventory/receiving', { target: '#main-content', pushUrl: true });
        return;
    }
    if (action === 'adjust-stock') {
        if (typeof window.toggleAdjustStockForm === 'function' && document.getElementById('adjust-stock-card')) {
            window.toggleAdjustStockForm();
            return;
        }
        htmx.ajax('GET', url || '/inventory/adjustments', { target: '#main-content', pushUrl: true });
        return;
    }
    if (action === 'add-item') {
        if (typeof window.toggleAddItemForm === 'function' && document.getElementById('add-item-card')) {
            window.toggleAddItemForm();
            return;
        }
        htmx.ajax('GET', '/items', { target: '#main-content', pushUrl: true });
        return;
    }
    if (action === 'add-brand') {
        if (typeof window.toggleAddBrandForm === 'function' && document.getElementById('add-brand-card')) {
            window.toggleAddBrandForm();
            return;
        }
        htmx.ajax('GET', '/brands', { target: '#main-content', pushUrl: true });
        return;
    }
    if (action === 'add-category') {
        if (typeof window.toggleAddCategoryForm === 'function' && document.getElementById('add-category-card')) {
            window.toggleAddCategoryForm();
            return;
        }
        htmx.ajax('GET', '/categories', { target: '#main-content', pushUrl: true });
        return;
    }
    if (action === 'add-uom-setting') {
        if (typeof window.toggleAddConversionForm === 'function' && document.getElementById('add-conversion-card')) {
            window.toggleAddConversionForm();
            return;
        }
        if (typeof window.toggleAddUomForm === 'function' && document.getElementById('add-uom-card')) {
            window.toggleAddUomForm();
            return;
        }
        if (typeof window.toggleAddUomSettingForm === 'function' && document.getElementById('add-uom-setting-card')) {
            window.toggleAddUomSettingForm();
            return;
        }
        htmx.ajax('GET', '/uoms', { target: '#main-content', pushUrl: true });
        return;
    }
    if (action === 'add-customer') {
        if (typeof window.toggleAddCustomerForm === 'function' && document.getElementById('add-customer-card')) {
            window.toggleAddCustomerForm();
            return;
        }
        htmx.ajax('GET', '/customers', { target: '#main-content', pushUrl: true });
        return;
    }
    if (action === 'record-sale') {
        if (typeof window.toggleEncodeSalesForm === 'function' && document.getElementById('encode-sales-card')) {
            window.toggleEncodeSalesForm();
            return;
        }
        htmx.ajax('GET', '/sales', { target: '#main-content', pushUrl: true });
        return;
    }
    if (action === 'add-goal') {
        const modal = document.getElementById('add-goal-modal');
        if (modal) {
            modal.style.display = 'flex';
            return;
        }
        htmx.ajax('GET', '/goals', { target: '#main-content', pushUrl: true });
        return;
    }
    if (action === 'add-reminder') {
        const modal = document.getElementById('add-reminder-modal');
        if (modal) {
            modal.style.display = 'flex';
            return;
        }
        htmx.ajax('GET', '/calendar?action=new', { target: '#main-content', pushUrl: true });
        return;
    }
    if (action === 'add-user') {
        const modal = document.getElementById('add-user-modal');
        if (modal) {
            modal.style.display = 'flex';
            return;
        }
        htmx.ajax('GET', '/settings/users', { target: '#main-content', pushUrl: true });
        return;
    }
    if (action === 'add-role') {
        const modal = document.getElementById('add-role-modal');
        if (modal) {
            modal.style.display = 'flex';
            return;
        }
        htmx.ajax('GET', '/settings/roles', { target: '#main-content', pushUrl: true });
        return;
    }
    if (url) {
        htmx.ajax('GET', url, { target: '#main-content', pushUrl: true });
    }
}
window.handleQuickAction = handleQuickAction;

function updateDynamicQuickMenu() {
    const path = window.location.pathname;
    const menu = document.getElementById("quick-create-menu");
    const defaultGrid = document.getElementById("quick-create-default-grid");
    const dynamicList = document.getElementById("quick-create-dynamic-list");
    const titleEl = document.getElementById("quick-create-header-title");
    if (!menu || !defaultGrid || !dynamicList) return;

    const isInventory = path === "/inventory" || path.startsWith("/inventory/");
    const isMasterlist = path === "/items" || path.startsWith("/items/") ||
        path === "/brands" || path.startsWith("/brands/") ||
        path === "/categories" || path.startsWith("/categories/") ||
        path === "/customers" || path.startsWith("/customers/") ||
        path === "/uoms" || path.startsWith("/uoms/") ||
        path === "/uom-settings" || path.startsWith("/uom-settings/");
    const isUsers = path.startsWith("/settings/users");
    const isRoles = path.startsWith("/settings/roles");
    const isSales = path === "/sales" || path.startsWith("/sales/");
    const isGoals = path === "/goals" || path.startsWith("/goals/");
    const isCalendar = path === "/calendar" || path.startsWith("/calendar/") ||
        path === "/reminders" || path.startsWith("/reminders/");

    const mobileBtn = document.querySelector(".quick-create-btn-mobile");

    if (isInventory) {
        menu.classList.add("dynamic-mode");
        defaultGrid.style.display = "none";
        dynamicList.style.display = "flex";
        if (mobileBtn) mobileBtn.textContent = "+ Stock";
        if (titleEl) titleEl.innerHTML = `<svg viewBox="0 0 24 24" width="13" height="13" stroke="hsl(243 75% 59%)" stroke-width="2.5" fill="none"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg> Stock Actions`;
        dynamicList.innerHTML = `
            <a href="/inventory/receiving" onclick="handleQuickAction('receive-stock', '/inventory/receiving', event)" class="quick-create-item">
                <div class="quick-create-item-icon receiving-color">
                    <svg viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2" fill="none"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg>
                </div>
                <div class="quick-create-item-text">
                    <span class="quick-create-item-name">+ Receive Stock</span>
                    <span class="quick-create-item-desc">Log vendor delivery</span>
                </div>
            </a>
            <a href="/inventory/adjustments" onclick="handleQuickAction('adjust-stock', '/inventory/adjustments', event)" class="quick-create-item">
                <div class="quick-create-item-icon adj-color">
                    <svg viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2" fill="none"><line x1="4" y1="21" x2="4" y2="14"></line><line x1="12" y1="21" x2="12" y2="12"></line><line x1="20" y1="21" x2="20" y2="16"></line></svg>
                </div>
                <div class="quick-create-item-text">
                    <span class="quick-create-item-name">+ Adjust Log</span>
                    <span class="quick-create-item-desc">Audit & correct count</span>
                </div>
            </a>
        `;
    } else if (isMasterlist && !isUsers && !isRoles) {
        menu.classList.add("dynamic-mode");
        defaultGrid.style.display = "none";
        dynamicList.style.display = "flex";
        if (mobileBtn) mobileBtn.textContent = "+ Masterlist";
        if (titleEl) titleEl.innerHTML = `<svg viewBox="0 0 24 24" width="13" height="13" stroke="hsl(243 75% 59%)" stroke-width="2.5" fill="none"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg> Masterlist Actions`;
        dynamicList.innerHTML = `
            <a href="/items" onclick="handleQuickAction('add-item', '/items', event)" class="quick-create-item">
                <div class="quick-create-item-icon item-color">
                    <svg viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2" fill="none"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path></svg>
                </div>
                <div class="quick-create-item-text">
                    <span class="quick-create-item-name">+ Add Item</span>
                    <span class="quick-create-item-desc">Catalog item & barcode</span>
                </div>
            </a>
            <a href="/brands" onclick="handleQuickAction('add-brand', '/brands', event)" class="quick-create-item">
                <div class="quick-create-item-icon settings-color">
                    <svg viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2" fill="none"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1.51 1V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg>
                </div>
                <div class="quick-create-item-text">
                    <span class="quick-create-item-name">+ Add Brand</span>
                    <span class="quick-create-item-desc">Brand classification</span>
                </div>
            </a>
            <a href="/categories" onclick="handleQuickAction('add-category', '/categories', event)" class="quick-create-item">
                <div class="quick-create-item-icon settings-color">
                    <svg viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2" fill="none"><rect x="3" y="3" width="7" height="7"></rect><rect x="14" y="3" width="7" height="7"></rect><rect x="14" y="14" width="7" height="7"></rect><rect x="3" y="14" width="7" height="7"></rect></svg>
                </div>
                <div class="quick-create-item-text">
                    <span class="quick-create-item-name">+ Add Category</span>
                    <span class="quick-create-item-desc">Product category</span>
                </div>
            </a>
            <a href="/uoms" onclick="handleQuickAction('add-uom-setting', '/uoms', event)" class="quick-create-item">
                <div class="quick-create-item-icon settings-color">
                    <svg viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2" fill="none"><path d="M4 7V4h3"></path><path d="M20 7V4h-3"></path><path d="M4 17v3h3"></path><path d="M20 17v3h-3"></path><rect x="7" y="7" width="10" height="10" rx="1"></rect></svg>
                </div>
                <div class="quick-create-item-text">
                    <span class="quick-create-item-name">+ Add UOM</span>
                    <span class="quick-create-item-desc">Units & conversion</span>
                </div>
            </a>
            <a href="/customers" onclick="handleQuickAction('add-customer', '/customers', event)" class="quick-create-item">
                <div class="quick-create-item-icon settings-color">
                    <svg viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2" fill="none"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><path d="M23 21v-2a4 4 0 0 0-3-3.87"></path><path d="M16 3.13a4 4 0 0 1 0 7.75"></path></svg>
                </div>
                <div class="quick-create-item-text">
                    <span class="quick-create-item-name">+ Add Customer</span>
                    <span class="quick-create-item-desc">Client directory & TIN</span>
                </div>
            </a>
        `;
    } else if (isSales) {
        menu.classList.add("dynamic-mode");
        defaultGrid.style.display = "none";
        dynamicList.style.display = "flex";
        if (mobileBtn) mobileBtn.textContent = "+ Sale";
        if (titleEl) titleEl.innerHTML = `<svg viewBox="0 0 24 24" width="13" height="13" stroke="hsl(243 75% 59%)" stroke-width="2.5" fill="none"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg> Sales Actions`;
        dynamicList.innerHTML = `
            <a href="/sales" onclick="handleQuickAction('record-sale', '/sales', event)" class="quick-create-item">
                <div class="quick-create-item-icon sales-color">
                    <svg viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2" fill="none"><line x1="12" y1="1" x2="12" y2="23"></line><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"></path></svg>
                </div>
                <div class="quick-create-item-text">
                    <span class="quick-create-item-name">+ Record Sale</span>
                    <span class="quick-create-item-desc">Customer invoice / receipt</span>
                </div>
            </a>
        `;
    } else if (isGoals) {
        menu.classList.add("dynamic-mode");
        defaultGrid.style.display = "none";
        dynamicList.style.display = "flex";
        if (mobileBtn) mobileBtn.textContent = "+ Goal";
        if (titleEl) titleEl.innerHTML = `<svg viewBox="0 0 24 24" width="13" height="13" stroke="hsl(243 75% 59%)" stroke-width="2.5" fill="none"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg> Earning Goals`;
        dynamicList.innerHTML = `
            <a href="/goals" onclick="handleQuickAction('add-goal', '/goals', event)" class="quick-create-item">
                <div class="quick-create-item-icon sales-color">
                    <svg viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2" fill="none"><circle cx="12" cy="12" r="10"></circle><circle cx="12" cy="12" r="6"></circle></svg>
                </div>
                <div class="quick-create-item-text">
                    <span class="quick-create-item-name">+ Set Earning Goal</span>
                    <span class="quick-create-item-desc">Set revenue target</span>
                </div>
            </a>
        `;
    } else if (isCalendar) {
        menu.classList.add("dynamic-mode");
        defaultGrid.style.display = "none";
        dynamicList.style.display = "flex";
        if (mobileBtn) mobileBtn.textContent = "+ Reminder";
        if (titleEl) titleEl.innerHTML = `<svg viewBox="0 0 24 24" width="13" height="13" stroke="hsl(243 75% 59%)" stroke-width="2.5" fill="none"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg> Calendar Actions`;
        dynamicList.innerHTML = `
            <a href="/calendar?action=new" onclick="handleQuickAction('add-reminder', '/calendar?action=new', event)" class="quick-create-item">
                <div class="quick-create-item-icon reminder-color">
                    <svg viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2" fill="none"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect><line x1="16" y1="2" x2="16" y2="6"></line><line x1="8" y1="2" x2="8" y2="6"></line><line x1="3" y1="10" x2="21" y2="10"></line></svg>
                </div>
                <div class="quick-create-item-text">
                    <span class="quick-create-item-name">+ Add Reminder</span>
                    <span class="quick-create-item-desc">Task deadline & alert</span>
                </div>
            </a>
        `;
    } else if (isUsers || isRoles) {
        menu.classList.add("dynamic-mode");
        defaultGrid.style.display = "none";
        dynamicList.style.display = "flex";
        if (mobileBtn) mobileBtn.textContent = "+ Admin";
        if (titleEl) titleEl.innerHTML = `<svg viewBox="0 0 24 24" width="13" height="13" stroke="hsl(243 75% 59%)" stroke-width="2.5" fill="none"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg> Admin Actions`;
        dynamicList.innerHTML = `
            <a href="/settings/users" onclick="handleQuickAction('add-user', '/settings/users', event)" class="quick-create-item">
                <div class="quick-create-item-icon settings-color">
                    <svg viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2" fill="none"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path><circle cx="12" cy="7" r="4"></circle></svg>
                </div>
                <div class="quick-create-item-text">
                    <span class="quick-create-item-name">+ Add User</span>
                    <span class="quick-create-item-desc">Create operator account</span>
                </div>
            </a>
            <a href="/settings/roles" onclick="handleQuickAction('add-role', '/settings/roles', event)" class="quick-create-item">
                <div class="quick-create-item-icon settings-color">
                    <svg viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2" fill="none"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1.51 1V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg>
                </div>
                <div class="quick-create-item-text">
                    <span class="quick-create-item-name">+ Create Role</span>
                    <span class="quick-create-item-desc">Role & permissions</span>
                </div>
            </a>
        `;
    } else {
        // Default: Show all buttons!
        menu.classList.remove("dynamic-mode");
        defaultGrid.style.display = "grid";
        dynamicList.style.display = "none";
        if (mobileBtn) mobileBtn.textContent = "+ New";
        if (titleEl) titleEl.innerHTML = `<svg viewBox="0 0 24 24" width="13" height="13" stroke="hsl(243 75% 59%)" stroke-width="2.5" fill="none"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg> Quick Actions`;
    }
}
window.updateDynamicQuickMenu = updateDynamicQuickMenu;

// Toggle QuickBooks "+ New" Quick Create Menu
function toggleQuickCreateMenu(event) {
    if (event) {
        event.preventDefault();
        event.stopPropagation();
    }
    const menu = document.getElementById("quick-create-menu");
    const btn = document.getElementById("quick-create-btn");
    if (!menu) return;
    const isOpen = menu.classList.toggle("show");
    if (btn) btn.setAttribute("aria-expanded", isOpen ? "true" : "false");
}

function closeQuickCreateMenu() {
    const menu = document.getElementById("quick-create-menu");
    const btn = document.getElementById("quick-create-btn");
    if (menu) menu.classList.remove("show");
    if (btn) btn.setAttribute("aria-expanded", "false");
    closeMobileSidebarOnNav();
}

document.addEventListener("DOMContentLoaded", updateActiveNavLink);
document.body.addEventListener("htmx:afterOnLoad", updateActiveNavLink);
document.body.addEventListener("htmx:historyRestore", updateActiveNavLink);
document.body.addEventListener("htmx:afterSwap", function (evt) {
    if (evt.detail.target && evt.detail.target.id === "main-content") {
        window.scrollTo(0, 0);
    }
});

// Helper function to show Toast notifications
function showToast(message, type) {
    type = type || 'info';
    message = message || '';

    const container = document.getElementById('toast-container');
    if (!container) return;

    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;

    // Icon based on type
    let icon = 'ℹ️';
    if (type === 'success') icon = '✅';
    if (type === 'error') icon = '❌';
    if (type === 'warning') icon = '⚠️';

    const iconSpan = document.createElement('span');
    iconSpan.style.marginRight = '0.5rem';
    iconSpan.textContent = icon;
    toast.appendChild(iconSpan);

    const textSpan = document.createElement('span');
    textSpan.textContent = message;
    toast.appendChild(textSpan);

    container.appendChild(toast);

    // Auto-remove toast
    setTimeout(() => {
        toast.style.animation = 'fadeOut 0.5s ease forwards';
        setTimeout(() => {
            toast.remove();
        }, 500);
    }, 4000);
}
window.showToast = showToast;

// Searchable Dropdown & Customer Autocomplete Caches
const _itemSearchHtmlCache = new Map();
const _customerSearchCache = new Map();

function clearSearchCaches() {
    _itemSearchHtmlCache.clear();
    _customerSearchCache.clear();
}
window.clearSearchCaches = clearSearchCaches;

// Toast notifications listener for HX-Trigger headers
document.body.addEventListener('show-toast', function (evt) {
    const type = (evt.detail && evt.detail.type) || 'info';
    const message = (evt.detail && evt.detail.message) || '';
    showToast(message, type);

    if (type === 'success') {
        const lowerMsg = message.toLowerCase();
        if (lowerMsg.includes('item') || lowerMsg.includes('customer') || lowerMsg.includes('uom') || lowerMsg.includes('brand') || lowerMsg.includes('category')) {
            clearSearchCaches();
        }
    }
});

// Global HTMX error handling
document.body.addEventListener('htmx:responseError', function (evt) {
    const xhr = evt.detail && evt.detail.xhr;
    const status = xhr ? xhr.status : 0;
    let msg = 'Request failed';
    if (xhr && xhr.responseText && xhr.responseText.length < 200 && !xhr.responseText.trim().startsWith('<')) {
        msg = xhr.responseText.trim();
    } else if (status) {
        msg = `Server error (${status}) — unable to complete request.`;
    }
    showToast(msg, 'error');
});

document.body.addEventListener('htmx:sendError', function () {
    showToast('Network error — unable to reach server.', 'error');
});

// Clear search caches on item/customer mutation triggers
['item-added', 'item-updated', 'customer-added', 'customer-updated', 'uom-added', 'uom-setting-added'].forEach(evtName => {
    document.body.addEventListener(evtName, clearSearchCaches);
});

// Searchable Dropdown functions
function updateDropdownPosition(input) {
    const container = input.closest('.searchable-dropdown');
    if (!container) return;
    const list = container.querySelector('.dropdown-list');
    if (!list || list.style.display === 'none') return;

    const rect = input.getBoundingClientRect();
    list.style.position = 'fixed';
    list.style.top = (rect.bottom + 2) + 'px';
    list.style.bottom = 'auto';
    list.style.left = rect.left + 'px';
    list.style.width = rect.width + 'px';
    list.style.zIndex = '999999';
    list.style.marginTop = '0px';
}

function cleanupDropdownState(container) {
    if (!container) return;
    container.classList.remove('active');
    const card = container.closest('.card');
    if (card) card.classList.remove('has-dropdown');
    const form = container.closest('form');
    if (form) form.classList.remove('has-dropdown');
    const tableContainer = container.closest('.table-container');
    if (tableContainer) tableContainer.classList.remove('has-dropdown');
    const row = container.closest('tr, .tr, .item-grid-row');
    if (row) row.classList.remove('active-dropdown-row');
    const cell = container.closest('td, .td');
    if (cell) cell.classList.remove('active-dropdown-cell');
}

function showDropdown(input) {
    const container = input.closest('.searchable-dropdown');
    if (!container) return;
    container.classList.add('active');
    const card = input.closest('.card');
    if (card) card.classList.add('has-dropdown');
    const form = input.closest('form');
    if (form) form.classList.add('has-dropdown');
    const tableContainer = input.closest('.table-container');
    if (tableContainer) tableContainer.classList.add('has-dropdown');
    const row = input.closest('tr, .tr, .item-grid-row');
    if (row) {
        document.querySelectorAll('tr.active-dropdown-row, .tr.active-dropdown-row, .item-grid-row.active-dropdown-row').forEach(r => r.classList.remove('active-dropdown-row'));
        row.classList.add('active-dropdown-row');
    }
    const cell = input.closest('td, .td');
    if (cell) {
        document.querySelectorAll('td.active-dropdown-cell, .td.active-dropdown-cell').forEach(c => c.classList.remove('active-dropdown-cell'));
        cell.classList.add('active-dropdown-cell');
    }
    const list = container.querySelector('.dropdown-list');
    if (list) {
        document.querySelectorAll('.dropdown-list').forEach(l => {
            if (l !== list) l.style.display = 'none';
        });
        list.style.display = 'block';
        updateDropdownPosition(input);
        filterDropdown(input);
    }
}

function hideDropdown(input) {
    setTimeout(() => {
        const container = input.closest('.searchable-dropdown');
        if (!container) return;
        cleanupDropdownState(container);
        const list = container.querySelector('.dropdown-list');
        if (list) {
            list.style.display = 'none';
            const hidden = container.querySelector('input[type="hidden"]');
            if (input.value.trim() === '') {
                if (hidden && hidden.value !== '') {
                    hidden.value = '';
                    if (window.htmx) {
                        htmx.trigger(hidden, 'change');
                    }
                    hidden.dispatchEvent(new Event('change', { bubbles: true }));
                }
            } else {
                const currentText = input.value.trim();
                const items = container.querySelectorAll('.dropdown-item');
                let match = null;
                for (const item of items) {
                    if ((item.getAttribute('data-description') || '').toLowerCase() === currentText.toLowerCase()) {
                        match = item;
                        break;
                    }
                }
                if (match) {
                    const selectedId = match.getAttribute('data-id');
                    input.value = match.getAttribute('data-description');
                    if (hidden && hidden.value !== selectedId) {
                        hidden.value = selectedId;
                        if (window.htmx) {
                            htmx.trigger(hidden, 'change');
                        }
                        hidden.dispatchEvent(new Event('change', { bubbles: true }));
                    }
                } else if (hidden && hidden.value) {
                    for (const item of items) {
                        if (item.getAttribute('data-id') === hidden.value) {
                            input.value = item.getAttribute('data-description');
                            break;
                        }
                    }
                }
            }
        }
    }, 200);
}

let _dropdownSearchTimer = null;
function filterDropdown(input) {
    const container = input.closest('.searchable-dropdown');
    if (!container) return;
    const hidden = container.querySelector('input[type="hidden"]');
    const isItemDropdown = hidden && (hidden.name === 'item_id' || hidden.id === 'item_id');

    // 1. Instant client-side filtering on items currently in DOM (0ms response)
    const filter = input.value.trim().toLowerCase();
    const items = container.querySelectorAll('.dropdown-item');
    let firstVisible = null;
    items.forEach(item => {
        const text = (item.getAttribute('data-description') || item.textContent || '').toLowerCase();
        const code = (item.getAttribute('data-code') || '').toLowerCase();
        if (filter === '' || text.includes(filter) || code.includes(filter)) {
            item.style.display = 'block';
            if (!firstVisible) firstVisible = item;
        } else {
            item.style.display = 'none';
        }
    });
    items.forEach(item => item.classList.remove('active'));
    if (firstVisible) {
        firstVisible.classList.add('active');
    }
    updateDropdownPosition(input);

    // 2. Supplement with full catalog fetch for item dropdowns
    if (isItemDropdown) {
        clearTimeout(_dropdownSearchTimer);
        const queryStr = input.value.trim();

        // If cached in memory, apply immediately with 0 network latency!
        if (_itemSearchHtmlCache.has(queryStr)) {
            const list = container.querySelector('.dropdown-list');
            if (list) {
                list.innerHTML = _itemSearchHtmlCache.get(queryStr);
                const first = list.querySelector('.dropdown-item');
                if (first) first.classList.add('active');
                updateDropdownPosition(input);
            }
            return;
        }

        // Rapid debounce (80ms) for background server fetch
        _dropdownSearchTimer = setTimeout(() => {
            fetch('/items/search?q=' + encodeURIComponent(queryStr) + '&limit=100')
                .then(res => {
                    if (!res.ok || res.redirected || (res.url && res.url.includes('/login'))) return null;
                    return res.text();
                })
                .then(html => {
                    if (!html) return;
                    if (_itemSearchHtmlCache.size >= 100) {
                        const firstKey = _itemSearchHtmlCache.keys().next().value;
                        _itemSearchHtmlCache.delete(firstKey);
                    }
                    _itemSearchHtmlCache.set(queryStr, html);
                    if (input.value.trim() === queryStr) {
                        const list = container.querySelector('.dropdown-list');
                        if (!list) return;
                        list.innerHTML = html;
                        const first = list.querySelector('.dropdown-item');
                        if (first) first.classList.add('active');
                        updateDropdownPosition(input);
                    }
                })
                .catch(err => console.error("Error querying items:", err));
        }, 80);
    }
}

function selectDropdownItem(item) {
    const container = item.closest('.searchable-dropdown');
    if (!container) return;
    const input = container.querySelector('.dropdown-search');
    const hidden = container.querySelector('input[type="hidden"]');

    const selectedId = item.getAttribute('data-id');
    const description = item.getAttribute('data-description');
    const uom = item.getAttribute('data-uom');

    if (input) input.value = description;

    let willSwap = false;
    if (hidden) {
        const oldVal = hidden.value;
        hidden.value = selectedId;
        if (oldVal !== selectedId || hidden.name === 'item_id') {
            if (hidden.name === 'item_id') {
                const itemRow = input.closest('#receiving-items-tbody tr, #adjustment-items-tbody tr, #sales-items-tbody tr, #scan-items-body tr, .item-grid-row, .scan-item-row');
                const tbody = itemRow ? itemRow.parentElement : null;
                if (tbody) {
                    willSwap = true;
                    const rowIndex = Array.from(tbody.children).indexOf(itemRow);
                    const handleRowSwap = function () {
                        tbody.removeEventListener('htmx:afterSwap', handleRowSwap);
                        setTimeout(() => {
                            const newRow = tbody.children[rowIndex];
                            if (newRow) {
                                const inputs = Array.from(newRow.querySelectorAll('input:not([type="hidden"]):not([readonly]):not([disabled]), select:not([readonly]):not([disabled])'));
                                if (inputs.length > 1) {
                                    const nextInput = inputs[1];
                                    nextInput.focus();
                                    if (nextInput.select && nextInput.type === 'text') nextInput.select();
                                }
                            }
                        }, 50);
                    };
                    tbody.addEventListener('htmx:afterSwap', handleRowSwap);
                    setTimeout(() => {
                        tbody.removeEventListener('htmx:afterSwap', handleRowSwap);
                    }, 2000);
                }
            }
            if (window.htmx) {
                htmx.trigger(hidden, 'change');
            }
            hidden.dispatchEvent(new Event('change', { bubbles: true }));
        }
    }

    if (uom) {
        const uomInput = container.closest('form')?.querySelector('#uom, #uom-inline-uom, [name="uom"]');
        if (uomInput) {
            uomInput.value = uom;
        }
    }

    const list = container.querySelector('.dropdown-list');
    if (list) list.style.display = 'none';
    cleanupDropdownState(container);

    if (!willSwap && input) {
        const itemRow = input.closest('#receiving-items-tbody tr, #adjustment-items-tbody tr, #sales-items-tbody tr, #scan-items-body tr, .item-grid-row, .scan-item-row');
        if (itemRow) {
            const inputs = Array.from(itemRow.querySelectorAll('input:not([type="hidden"]):not([readonly]):not([disabled]), select:not([readonly]):not([disabled])'));
            const index = inputs.indexOf(input);
            if (index !== -1 && index < inputs.length - 1) {
                const nextInput = inputs[index + 1];
                nextInput.focus();
                if (nextInput.select && nextInput.type === 'text') {
                    nextInput.select();
                }
            }
        }
    }
}

function handleDropdownKeydown(event, input) {
    const container = input.closest('.searchable-dropdown');
    if (!container) return;
    const list = container.querySelector('.dropdown-list');
    if (!list || list.style.display === 'none') {
        if (event.key === 'ArrowDown') {
            showDropdown(input);
            event.preventDefault();
        }
        return;
    }

    const visibleItems = Array.from(container.querySelectorAll('.dropdown-item')).filter(item => item.style.display !== 'none');
    if (visibleItems.length === 0) return;

    let activeIndex = visibleItems.findIndex(item => item.classList.contains('active'));

    if (event.key === 'ArrowDown') {
        event.preventDefault();
        if (activeIndex >= 0) visibleItems[activeIndex].classList.remove('active');
        activeIndex = (activeIndex + 1) % visibleItems.length;
        visibleItems[activeIndex].classList.add('active');
        visibleItems[activeIndex].scrollIntoView({ block: 'nearest' });
    } else if (event.key === 'ArrowUp') {
        event.preventDefault();
        if (activeIndex >= 0) visibleItems[activeIndex].classList.remove('active');
        activeIndex = (activeIndex - 1 + visibleItems.length) % visibleItems.length;
        visibleItems[activeIndex].classList.add('active');
        visibleItems[activeIndex].scrollIntoView({ block: 'nearest' });
    } else if (event.key === 'Enter') {
        event.preventDefault();
        event.stopPropagation();
        if (activeIndex >= 0) {
            selectDropdownItem(visibleItems[activeIndex]);
        } else {
            selectDropdownItem(visibleItems[0]);
        }
    } else if (event.key === 'Escape') {
        event.preventDefault();
        list.style.display = 'none';
        cleanupDropdownState(container);
        input.blur();
    }
}

// Global click listener to close dropdowns when clicking outside
document.addEventListener('click', function (event) {
    if (!event.target.closest('.searchable-dropdown')) {
        document.querySelectorAll('.dropdown-list').forEach(list => {
            list.style.display = 'none';
        });
        document.querySelectorAll('.searchable-dropdown.active').forEach(c => {
            cleanupDropdownState(c);
        });
    }
});

// Customer Autocomplete and Autofill Integration
let _custSearchDebounceTimer = null;

function escapeCustHtml(str) {
    if (!str) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

function handleCustomerAutocomplete(inputEl, tinInputId, addressInputId, contactInputId, termsSelectId) {
    if (!inputEl) return;
    const query = inputEl.value.trim();
    const parent = inputEl.parentElement;
    if (!parent) return;
    let dropdown = parent.querySelector('.customer-autocomplete-dropdown, .dropdown-list');
    if (!dropdown) {
        dropdown = document.createElement('div');
        dropdown.className = 'dropdown-list customer-autocomplete-dropdown';
        dropdown.style.cssText = 'display: none; position: absolute; top: calc(100% + 4px); left: 0; right: 0; z-index: 1050; max-height: 220px; overflow-y: auto; background: var(--bg-surface, #fff); border: 1px solid var(--border-color); border-radius: var(--radius-md, 8px); box-shadow: 0 10px 15px -3px rgba(0,0,0,0.1), 0 4px 6px -4px rgba(0,0,0,0.1);';
        parent.appendChild(dropdown);
    }

    if (query.length === 0) {
        dropdown.style.display = 'none';
        dropdown.innerHTML = '';
        return;
    }

    const renderResults = (customers) => {
        if (!Array.isArray(customers) || customers.length === 0) {
            dropdown.innerHTML = '<div style="padding: 0.5rem 0.75rem; font-size: 0.8125rem; color: var(--text-muted);">No matching customers in directory</div>';
            dropdown.style.display = 'block';
            return;
        }

        dropdown.innerHTML = customers.map((c, idx) => {
            const cName = c.name || c.Name || '';
            const cCode = c.code || c.Code || '';
            const cTin = c.tin_no || c.TINNo || '';
            const cAddr = c.address || c.Address || '';
            const cContact = c.contact_person || c.ContactPerson || '';
            const cTerms = c.payment_terms || c.PaymentTerms || '';
            return `
            <div class="dropdown-item ${idx === 0 ? 'active' : ''}" 
                 style="padding: 0.5rem 0.75rem; cursor: pointer; border-bottom: 1px solid var(--border-color, #eee);"
                 data-name="${escapeCustHtml(cName)}"
                 data-tin="${escapeCustHtml(cTin)}"
                 data-address="${escapeCustHtml(cAddr)}"
                 data-contact="${escapeCustHtml(cContact)}"
                 data-terms="${escapeCustHtml(cTerms)}"
                 onmousedown="applyCustomerSelection(this, '${inputEl.id}', '${tinInputId || ''}', '${addressInputId || ''}', '${contactInputId || ''}', '${termsSelectId || ''}')">
                <div style="font-weight: 600; font-size: 0.8125rem; color: var(--text-main);">${escapeCustHtml(cName)}</div>
                <div style="font-size: 0.75rem; color: var(--text-muted); display: flex; gap: 0.5rem; margin-top: 2px;">
                    <span style="font-family: monospace; background: var(--bg-surface-elevated, #f1f5f9); padding: 0 4px; border-radius: 3px;">${escapeCustHtml(cCode)}</span>
                    ${cTin ? `<span>• TIN: ${escapeCustHtml(cTin)}</span>` : ''}
                    ${cAddr ? `<span>• ${escapeCustHtml(cAddr.length > 30 ? cAddr.slice(0, 30) + '...' : cAddr)}</span>` : ''}
                </div>
            </div>
        `;
        }).join('');
        dropdown.style.display = 'block';
    };

    if (_customerSearchCache.has(query)) {
        renderResults(_customerSearchCache.get(query));
        return;
    }

    clearTimeout(_custSearchDebounceTimer);
    _custSearchDebounceTimer = setTimeout(() => {
        fetch('/customers/search?q=' + encodeURIComponent(query) + '&limit=20&format=json', {
            headers: { 'Accept': 'application/json' }
        })
            .then(res => {
                if (!res.ok || res.redirected || (res.url && res.url.includes('/login'))) return null;
                return res.json();
            })
            .then(data => {
                if (Array.isArray(data)) {
                    if (_customerSearchCache.size >= 100) {
                        const firstKey = _customerSearchCache.keys().next().value;
                        _customerSearchCache.delete(firstKey);
                    }
                    _customerSearchCache.set(query, data);
                    if (inputEl.value.trim() === query) {
                        renderResults(data);
                    }
                }
            })
            .catch(err => console.error("Customer search error:", err));
    }, 80);
}

function handleCustomerKeydown(event, inputEl, tinInputId, addressInputId, contactInputId, termsSelectId) {
    const parent = inputEl.parentElement;
    if (!parent) return;
    const dropdown = parent.querySelector('.customer-autocomplete-dropdown, .dropdown-list');
    if (!dropdown || dropdown.style.display === 'none') return;

    const items = Array.from(dropdown.querySelectorAll('.dropdown-item'));
    if (items.length === 0) return;

    let activeIdx = items.findIndex(el => el.classList.contains('active'));

    if (event.key === 'ArrowDown') {
        event.preventDefault();
        if (activeIdx >= 0) items[activeIdx].classList.remove('active');
        activeIdx = (activeIdx + 1) % items.length;
        items[activeIdx].classList.add('active');
        items[activeIdx].scrollIntoView({ block: 'nearest' });
    } else if (event.key === 'ArrowUp') {
        event.preventDefault();
        if (activeIdx >= 0) items[activeIdx].classList.remove('active');
        activeIdx = (activeIdx - 1 + items.length) % items.length;
        items[activeIdx].classList.add('active');
        items[activeIdx].scrollIntoView({ block: 'nearest' });
    } else if (event.key === 'Enter') {
        event.preventDefault();
        if (activeIdx >= 0 && items[activeIdx]) {
            applyCustomerSelection(items[activeIdx], inputEl.id, tinInputId, addressInputId, contactInputId, termsSelectId);
        }
    } else if (event.key === 'Escape') {
        dropdown.style.display = 'none';
    }
}

function applyCustomerSelection(itemEl, nameInputId, tinInputId, addressInputId, contactInputId, termsSelectId) {
    const name = itemEl.getAttribute('data-name') || '';
    const tin = itemEl.getAttribute('data-tin') || '';
    const address = itemEl.getAttribute('data-address') || '';
    const contact = itemEl.getAttribute('data-contact') || '';
    const terms = itemEl.getAttribute('data-terms') || '';

    if (nameInputId) {
        const nameEl = document.getElementById(nameInputId);
        if (nameEl) {
            nameEl.value = name;
            nameEl.dispatchEvent(new Event('input', { bubbles: true }));
            nameEl.dispatchEvent(new Event('change', { bubbles: true }));
        }
    }
    if (tinInputId) {
        const tinEl = document.getElementById(tinInputId);
        if (tinEl) {
            tinEl.value = tin;
            tinEl.dispatchEvent(new Event('input', { bubbles: true }));
        }
    }
    if (addressInputId) {
        const addrEl = document.getElementById(addressInputId);
        if (addrEl) {
            addrEl.value = address;
            addrEl.dispatchEvent(new Event('input', { bubbles: true }));
        }
    }
    if (contactInputId) {
        const contEl = document.getElementById(contactInputId);
        if (contEl) {
            contEl.value = contact;
            contEl.dispatchEvent(new Event('input', { bubbles: true }));
        }
    }
    if (termsSelectId && terms) {
        const termsEl = document.getElementById(termsSelectId);
        if (termsEl) {
            for (let opt of termsEl.options) {
                if (opt.value.toLowerCase().includes(terms.toLowerCase()) || terms.toLowerCase().includes(opt.value.toLowerCase())) {
                    termsEl.value = opt.value;
                    termsEl.dispatchEvent(new Event('change', { bubbles: true }));
                    break;
                }
            }
        }
    }

    const dropdown = itemEl.closest('.dropdown-list');
    if (dropdown) dropdown.style.display = 'none';
}

window.handleCustomerAutocomplete = handleCustomerAutocomplete;
window.handleCustomerKeydown = handleCustomerKeydown;
window.applyCustomerSelection = applyCustomerSelection;

window.addEventListener('scroll', function () {
    const activeInput = document.activeElement;
    if (activeInput && activeInput.classList && activeInput.classList.contains('dropdown-search')) {
        updateDropdownPosition(activeInput);
    }
}, true);

window.addEventListener('resize', function () {
    const activeInput = document.activeElement;
    if (activeInput && activeInput.classList && activeInput.classList.contains('dropdown-search')) {
        updateDropdownPosition(activeInput);
    }
});

// Global keydown handler for Enter key behavior:
// 1. On filter inputs/bars, pressing Enter applies the filter.
// 2. When adding items in receiving, adjustments, or sales, pressing Enter adds another item line.
document.addEventListener('keydown', function (e) {
    if (e.key !== 'Enter') return;
    const target = e.target;
    if (!target || !['INPUT', 'SELECT', 'TEXTAREA'].includes(target.tagName)) return;

    // If a searchable dropdown list is currently open/visible, let handleDropdownKeydown handle Enter
    const dropdown = target.closest('.searchable-dropdown');
    if (dropdown) {
        const list = dropdown.querySelector('.dropdown-list');
        if (list && list.style.display !== 'none' && list.style.display !== '') {
            return;
        }
    }

    // Check if we are inside an item addition form or item row
    const itemForm = target.closest('form[hx-post*="/receiving/add"], form[hx-post*="/adjustments/add"], form[hx-post*="/sales/add"], #receipt-scan-form');
    const isItemTable = target.closest('#receiving-items-tbody, #adjustment-items-tbody, #sales-items-tbody, #scan-items-body');

    // 1. FILTER BAR / FILTER INPUTS -> Apply filter
    const filterContainer = target.closest('.search-filter-bar, form[id*="filter"], form[onsubmit*="return false"]');
    const isFilterInput = filterContainer || target.name === 'search' || (target.name && target.name.includes('filter')) || target.hasAttribute('data-filter');

    if (isFilterInput && !itemForm && !isItemTable) {
        e.preventDefault();
        const container = filterContainer || target.closest('form, div') || target.parentElement;
        if (container) {
            const hxTarget = target.hasAttribute('hx-get') ? target : (container.querySelector('[hx-get]') || target);
            if (window.htmx && hxTarget && hxTarget.hasAttribute('hx-get')) {
                htmx.trigger(hxTarget, 'search');
                htmx.trigger(hxTarget, 'change');
                htmx.trigger(hxTarget, 'keyup');
            } else if (target.form && target.form.onsubmit !== false) {
                target.form.submit();
            }
        }
        return;
    }

    // 2. ADDING ITEMS IN RECEIVING, ADJUSTMENTS, OR SALES -> Enter navigates or adds a row
    if (itemForm || isItemTable) {
        e.preventDefault();
        const itemRow = target.closest('#receiving-items-tbody tr, #adjustment-items-tbody tr, #sales-items-tbody tr, #scan-items-body tr, .item-grid-row, .scan-item-row');
        if (itemRow) {
            const inputs = Array.from(itemRow.querySelectorAll('input:not([type="hidden"]):not([readonly]):not([disabled]), select:not([readonly]):not([disabled])'));
            const index = inputs.indexOf(target);
            if (index !== -1) {
                if (index < inputs.length - 1) {
                    // Go to the next input in the current row
                    const nextInput = inputs[index + 1];
                    nextInput.focus();
                    if (nextInput.select && nextInput.type === 'text') {
                        nextInput.select();
                    }
                } else {
                    // We are at the last input, add a new itemline
                    const parent = itemForm || target.closest('.card, form, div');
                    if (parent) {
                        const addRowBtn = parent.querySelector('button[hx-get*="new-row"], button[onclick*="addScanRow"], button[onclick*="addRow"]') ||
                            Array.from(parent.querySelectorAll('button')).find(btn => btn.textContent.includes('Add Item Row') || btn.textContent.includes('Add Row'));
                        if (addRowBtn) {
                            const isHtmx = addRowBtn.hasAttribute('hx-get') || addRowBtn.hasAttribute('hx-post');
                            const tbody = parent.querySelector('tbody, #sales-items-tbody, #receiving-items-tbody, #adjustment-items-tbody, .item-grid-container > div');
                            if (isHtmx && tbody) {
                                const handleSwap = function () {
                                    tbody.removeEventListener('htmx:afterSwap', handleSwap);
                                    setTimeout(() => {
                                        if (tbody.lastElementChild) {
                                            const newInput = tbody.lastElementChild.querySelector('.dropdown-search, input:not([type="hidden"]), select');
                                            if (newInput) {
                                                newInput.focus();
                                                if (newInput.select && newInput.type === 'text') newInput.select();
                                            }
                                        }
                                    }, 50);
                                };
                                tbody.addEventListener('htmx:afterSwap', handleSwap);
                                // Fallback timeout in case of error
                                setTimeout(() => {
                                    tbody.removeEventListener('htmx:afterSwap', handleSwap);
                                }, 2000);
                            }

                            addRowBtn.click();

                            if (!isHtmx && tbody) {
                                setTimeout(() => {
                                    if (tbody.lastElementChild) {
                                        const newInput = tbody.lastElementChild.querySelector('.dropdown-search, input:not([type="hidden"]), select');
                                        if (newInput) {
                                            newInput.focus();
                                            if (newInput.select && newInput.type === 'text') newInput.select();
                                        }
                                    }
                                }, 50);
                            }
                        }
                    }
                }
            }
        }
        return;
    }
});

// Global shadcn/ui Alert Dialog Handler
window.showShadcnAlertDialog = function (options) {
    return new Promise((resolve) => {
        const dialog = document.getElementById('shadcn-alert-dialog');
        const titleEl = document.getElementById('shadcn-alert-dialog-title');
        const descEl = document.getElementById('shadcn-alert-dialog-description');
        const cancelBtn = document.getElementById('shadcn-alert-dialog-cancel');
        const actionBtn = document.getElementById('shadcn-alert-dialog-action');

        if (!dialog || !titleEl || !descEl || !cancelBtn || !actionBtn) {
            if (options.isConfirm) {
                const res = window.nativeConfirm ? window.nativeConfirm(options.description) : confirm(options.description);
                if (options.onConfirm && res) options.onConfirm();
                if (options.onCancel && !res) options.onCancel();
                resolve(res);
            } else {
                window.nativeAlert ? window.nativeAlert(options.description) : alert(options.description);
                if (options.onAction) options.onAction();
                resolve(true);
            }
            return;
        }

        titleEl.textContent = options.title || (options.isConfirm ? 'Are you sure?' : 'Alert');
        descEl.textContent = options.description || '';

        if (options.isConfirm || options.showCancel) {
            cancelBtn.style.display = 'inline-flex';
            cancelBtn.textContent = options.cancelText || 'Cancel';
        } else {
            cancelBtn.style.display = 'none';
        }

        actionBtn.textContent = options.actionText || options.confirmText || (options.isConfirm ? 'Continue' : 'OK');

        const isDestructive = options.isDestructive ||
            (options.description && (options.description.toLowerCase().includes('delete') || options.description.toLowerCase().includes('remove')));

        actionBtn.className = isDestructive ? 'btn btn-danger' : 'btn btn-primary';

        function close(confirmed) {
            dialog.style.display = 'none';
            cleanup();
            if (confirmed) {
                if (options.onConfirm) options.onConfirm();
                if (options.onAction) options.onAction();
                resolve(true);
            } else {
                if (options.onCancel) options.onCancel();
                resolve(false);
            }
        }

        function handleCancel() { close(false); }
        function handleAction() { close(true); }
        function handleKeydown(e) {
            if (e.key === 'Escape') {
                e.preventDefault();
                close(false);
            } else if (e.key === 'Enter') {
                e.preventDefault();
                close(true);
            }
        }
        function handleBackdropClick(e) {
            if (e.target === dialog) {
                close(false);
            }
        }

        function cleanup() {
            cancelBtn.removeEventListener('click', handleCancel);
            actionBtn.removeEventListener('click', handleAction);
            document.removeEventListener('keydown', handleKeydown);
            dialog.removeEventListener('click', handleBackdropClick);
        }

        cancelBtn.addEventListener('click', handleCancel);
        actionBtn.addEventListener('click', handleAction);
        document.addEventListener('keydown', handleKeydown);
        dialog.addEventListener('click', handleBackdropClick);

        dialog.style.display = 'flex';
        actionBtn.focus();
    });
};

// Intercept HTMX hx-confirm event
document.addEventListener('htmx:confirm', function (evt) {
    const question = evt.detail.question;
    // Only intercept when there's an actual hx-confirm question; otherwise let HTMX proceed normally
    if (!question) return;
    evt.preventDefault();
    showShadcnAlertDialog({
        title: 'Are you sure?',
        description: question,
        isConfirm: true,
        confirmText: 'Continue',
        cancelText: 'Cancel',
        isDestructive: question.toLowerCase().includes('delete') || question.toLowerCase().includes('remove'),
        onConfirm: function () {
            evt.detail.issueRequest(true);
        }
    });
});

// Override standard window.alert
if (!window.nativeAlert) {
    window.nativeAlert = window.alert;
}
window.alert = function (message) {
    showShadcnAlertDialog({
        title: 'Alert',
        description: message,
        isConfirm: false,
        actionText: 'OK'
    });
};

// Expose async confirm dialog helper while leaving window.confirm synchronous
if (!window.nativeConfirm) {
    window.nativeConfirm = window.confirm;
}
window.confirmAsync = function (message, title) {
    return showShadcnAlertDialog({
        title: title || 'Confirm Action',
        description: message,
        isConfirm: true,
        confirmText: 'Confirm',
        cancelText: 'Cancel'
    });
};
