// UOM Setting Modal helpers
window._uomModalSourceRow = null;

window.openUomSettingModal = function (itemId, itemDesc) {
    document.getElementById('uom-modal-item-id').value = itemId;
    document.getElementById('uom-modal-subtitle').textContent = itemDesc ? 'Item: ' + itemDesc : '';
    document.getElementById('uom-modal-muom').value = '';
    document.getElementById('uom-modal-factor').value = '';
    // Remember which item row triggered this (for refresh after save)
    window._uomModalSourceRow = document.querySelector('.item-grid-row input[name="item_id"][value="' + itemId + '"]');
    const modal = document.getElementById('uom-setting-modal');
    modal.style.display = 'flex';
    setTimeout(() => document.getElementById('uom-modal-muom').focus(), 50);
};

window.closeUomSettingModal = function () {
    document.getElementById('uom-setting-modal').style.display = 'none';
    window._uomModalSourceRow = null;
};

window.handleUomModalAfterRequest = function (event) {
    if (event.detail.successful) {
        closeUomSettingModal();
        // Re-trigger the item row's hidden input to reload the UOM dropdown
        if (window._uomModalSourceRow) {
            htmx.trigger(window._uomModalSourceRow, 'change');
        }
    }
};

// Close modal when clicking the backdrop
const _uomModalEl = document.getElementById('uom-setting-modal');
if (_uomModalEl) {
    _uomModalEl.addEventListener('click', function (e) {
        if (e.target === this) closeUomSettingModal();
    });
}

// Close modal on Escape key
document.addEventListener('keydown', function (e) {
    const modal = document.getElementById('uom-setting-modal');
    if (e.key === 'Escape' && modal && modal.style.display === 'flex') {
        closeUomSettingModal();
    }
});

// =========================================================================
// Universal List Views Engine:
// 1. Customizable Column Header Colors (Table Themes + Column Accents)
// 2. Smooth 60-FPS Column Resizing with Double-Click Reset
// 3. Semantic Column Grouping & Group Hide/Unhide
// 4. Column Reordering (Header Drag & Drop + Popover Controls)
// 5. Per-User Database Persistence (/api/user-preferences + localStorage)
// =========================================================================

function escapeTableHtml(str) {
    if (!str) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

window._radlinePrefTimers = window._radlinePrefTimers || {};
window.saveRadlinePreference = function (key, valueStr, isDelete) {
    try {
        if (isDelete || !valueStr) {
            localStorage.removeItem(key);
        } else {
            localStorage.setItem(key, valueStr);
        }
    } catch (e) { }

    if (window._radlinePrefTimers[key]) {
        clearTimeout(window._radlinePrefTimers[key]);
    }
    window._radlinePrefTimers[key] = setTimeout(() => {
        fetch('/api/user-preferences', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                key: key,
                value: isDelete ? '' : (valueStr || ''),
                delete: !!isDelete
            })
        }).catch(() => { });
    }, 250);
};

window.syncRadlinePreferencesFromServer = function () {
    if (window._radlinePrefsSynced) return;
    window._radlinePrefsSynced = true;
    fetch('/api/user-preferences', { headers: { 'Accept': 'application/json' } })
        .then(r => r.ok ? r.json() : null)
        .then(data => {
            if (!data || !data.preferences) return;
            let changed = false;
            Object.keys(data.preferences).forEach(k => {
                if (k.startsWith('radline_') || k.startsWith('rav_')) {
                    const srvVal = data.preferences[k];
                    if (srvVal && localStorage.getItem(k) !== srvVal) {
                        localStorage.setItem(k, srvVal);
                        changed = true;
                    }
                }
            });
            if (changed && typeof window.initExcelCollapsibleColumns === 'function') {
                window.initExcelCollapsibleColumns();
            }
        })
        .catch(() => { });
};

window.RADLINE_TABLE_GROUPS = {
    'sales-log-table': [
        { id: 'doc_info', label: 'Document Info', cols: ['doc_status', 'doc_date', 'doc_type_number'] },
        { id: 'customer_info', label: 'Customer Info', cols: ['customer_name', 'tin_no', 'address'] },
        { id: 'product_info', label: 'Product Info', cols: ['supplier', 'item_description', 'qty', 'uom'] },
        { id: 'financials', label: 'Financials & Remittance', cols: ['price', 'total_sales', 'cost', 'total_cost', 'patong', 'total_remit', 'profit', 'profit_margin', 'remarks'] }
    ],
    'inventory-stock-table': [
        { id: 'item_details', label: 'Item Details', cols: ['description', 'uom'] },
        { id: 'stock_movements', label: 'Stock Movements', cols: ['received', 'sold', 'adjusted', 'on_hand'] },
        { id: 'pricing_cost', label: 'Pricing & Cost', cols: ['current_cost', 'current_price'] }
    ],
    'monthly-inventory-table': [
        { id: 'item_period', label: 'Item & Period', cols: ['month', 'description', 'uom'] },
        { id: 'monthly_movements', label: 'Monthly Movements', cols: ['received', 'sold', 'adjusted', 'net_change'] }
    ],
    'items-table': [
        { id: 'item_identity', label: 'Item Details', cols: ['description', 'model'] },
        { id: 'item_classification', label: 'Classification & UOM', cols: ['brand', 'category', 'default_uom'] }
    ],
    'receiving-logs-table': [
        { id: 'rec_doc', label: 'Document Info', cols: ['pl_id', 'date', 'supplier'] },
        { id: 'rec_item', label: 'Item & Valuation', cols: ['item_description', 'qty', 'uom', 'cost_per_unit', 'total_cost', 'remarks'] }
    ],
    'adjustment-logs-table': [
        { id: 'adj_doc', label: 'Adjustment Info', cols: ['adj_id', 'date', 'item_description'] },
        { id: 'adj_impact', label: 'Quantity & Cost Impact', cols: ['adj_qty', 'uom', 'cost_impact', 'remarks'] }
    ]
};

window.RADLINE_HEADER_THEMES = [
    { id: 'slate', label: 'Slate', dot: '#cbd5e1' },
    { id: 'indigo', label: 'Indigo', dot: '#6366f1' },
    { id: 'emerald', label: 'Emerald', dot: '#10b981' },
    { id: 'amber', label: 'Amber', dot: '#f59e0b' },
    { id: 'navy', label: 'Navy', dot: '#1e293b' },
    { id: 'rose', label: 'Rose', dot: '#f43f5e' }
];

window.RADLINE_COL_ACCENTS = [
    { id: '', label: 'None', dot: '#f8fafc' },
    { id: 'indigo', label: 'Indigo', dot: '#6366f1' },
    { id: 'emerald', label: 'Emerald', dot: '#10b981' },
    { id: 'amber', label: 'Amber', dot: '#f59e0b' },
    { id: 'rose', label: 'Rose', dot: '#f43f5e' },
    { id: 'purple', label: 'Purple', dot: '#a855f7' },
    { id: 'sky', label: 'Sky', dot: '#0ea5e9' }
];

function getWidthStorageKey(tableId) {
    if (tableId === 'sales-log-table') return 'radline_sales_column_widths';
    if (tableId === 'inventory-stock-table') return 'radline_inventory_column_widths';
    if (tableId === 'monthly-inventory-table') return 'radline_monthly_inventory_column_widths';
    if (tableId === 'items-table') return 'radline_items_column_widths';
    if (tableId === 'receiving-logs-table') return 'radline_receiving_column_widths';
    if (tableId === 'adjustment-logs-table') return 'radline_adjustment_column_widths';
    return 'radline_col_widths_' + tableId;
}

function getPopoverForTable(tableId) {
    if (tableId === 'sales-log-table') return document.getElementById('customize-view-popover');
    if (tableId === 'inventory-stock-table') return document.getElementById('customize-inventory-view-popover');
    if (tableId === 'monthly-inventory-table') return document.getElementById('customize-monthly-view-popover');
    if (tableId === 'items-table') return document.getElementById('customize-items-view-popover');
    return null;
}

window.resetRadlineTableCustomization = function (tableId) {
    const colorKey = 'radline_header_colors_' + tableId;
    const orderKey = 'radline_col_order_' + tableId;
    const collapseKey = 'radline_collapsed_cols_' + tableId;
    const widthKey = getWidthStorageKey(tableId);

    window.saveRadlinePreference(colorKey, '', true);
    window.saveRadlinePreference(orderKey, '', true);
    window.saveRadlinePreference(collapseKey, '', true);
    window.saveRadlinePreference(widthKey, '', true);

    const styleEl = document.getElementById('radline-universal-widths-' + tableId);
    if (styleEl) styleEl.textContent = '';

    const table = document.getElementById(tableId);
    if (table && table._defaultColOrder) {
        applyColumnOrderToDOM(table, table._defaultColOrder);
    }
    if (typeof window.initExcelCollapsibleColumns === 'function') {
        window.initExcelCollapsibleColumns();
    }
};

function applyColumnOrderToDOM(table, colOrder) {
    if (!colOrder || !colOrder.length) return;
    const theadRow = table.querySelector('.thead .tr, thead tr');
    if (theadRow) {
        const thMap = {};
        Array.from(theadRow.children).forEach(th => {
            const c = th.getAttribute('data-col');
            if (c) thMap[c] = th;
        });
        colOrder.forEach(c => {
            if (thMap[c]) theadRow.appendChild(thMap[c]);
        });
        if (thMap['actions']) theadRow.appendChild(thMap['actions']);
    }
    table.querySelectorAll('.tbody .tr, tbody tr').forEach(row => {
        const tdMap = {};
        Array.from(row.children).forEach(td => {
            const c = td.getAttribute('data-col');
            if (c) tdMap[c] = td;
        });
        if (Object.keys(tdMap).length <= 1) return;
        colOrder.forEach(c => {
            if (tdMap[c]) row.appendChild(tdMap[c]);
        });
        if (tdMap['actions']) row.appendChild(tdMap['actions']);
    });
}

window.initExcelCollapsibleColumns = function () {
    window.syncRadlinePreferencesFromServer();

    const tables = document.querySelectorAll('.table, .resizable-table, .sales-log-table, table.table-scroll-md, table.table-scroll-lg, .excel-table');
    tables.forEach((table, tableIdx) => {
        const tableId = table.id || table.getAttribute('data-table-id') ||
            (table.classList.contains('inventory-stock-table') ? 'inventory-stock-table' :
                (table.classList.contains('sales-log-table') ? 'sales-log-table' :
                    (table.classList.contains('items-table') ? 'items-table' :
                        (table.classList.contains('monthly-inventory-table') ? 'monthly-inventory-table' :
                            (table.classList.contains('receiving-logs-table') ? 'receiving-logs-table' :
                                (table.classList.contains('adjustment-logs-table') ? 'adjustment-logs-table' :
                                    ('radline_table_' + tableIdx)))))));
        if (!table.id) table.id = tableId;

        const storageKey = 'radline_collapsed_cols_' + tableId;
        const colorKey = 'radline_header_colors_' + tableId;
        const orderKey = 'radline_col_order_' + tableId;
        const widthKey = getWidthStorageKey(tableId);

        let collapsedCols = [];
        try {
            collapsedCols = JSON.parse(localStorage.getItem(storageKey) || '[]');
        } catch (e) {
            collapsedCols = [];
        }

        let colorConfig = { theme: 'slate', accents: {} };
        try {
            const parsedColors = JSON.parse(localStorage.getItem(colorKey) || 'null');
            if (parsedColors && typeof parsedColors === 'object') {
                colorConfig.theme = parsedColors.theme || 'slate';
                colorConfig.accents = parsedColors.accents || {};
            }
        } catch (e) { }

        let colWidths = {};
        try {
            colWidths = JSON.parse(localStorage.getItem(widthKey) || '{}') || {};
        } catch (e) {
            colWidths = {};
        }

        // Assign data-col to headers & cells if missing
        const initialHeaders = Array.from(table.querySelectorAll('.thead .th, thead th'));
        initialHeaders.forEach((th, idx) => {
            let colKey = th.getAttribute('data-col');
            if (!colKey) {
                colKey = 'col_' + idx;
                th.setAttribute('data-col', colKey);
                table.querySelectorAll('tbody tr, .tbody .tr').forEach(row => {
                    if (row.children[idx]) row.children[idx].setAttribute('data-col', colKey);
                });
            }
        });

        if (!table._defaultColOrder) {
            table._defaultColOrder = initialHeaders
                .map(h => h.getAttribute('data-col'))
                .filter(c => c && c !== 'actions');
        }

        let colOrder = [...table._defaultColOrder];
        try {
            const savedOrder = JSON.parse(localStorage.getItem(orderKey) || 'null');
            if (Array.isArray(savedOrder) && savedOrder.length > 0) {
                const validSaved = savedOrder.filter(c => table._defaultColOrder.includes(c));
                const missing = table._defaultColOrder.filter(c => !validSaved.includes(c));
                colOrder = [...validSaved, ...missing];
            }
        } catch (e) { }

        // Apply column order to DOM
        applyColumnOrderToDOM(table, colOrder);

        // Re-query headers in their current DOM order
        const getOrderedHeaders = () => Array.from(table.querySelectorAll('.thead .th, thead th'));

        // Apply Header Theme & Column Accent Colors
        const applyColors = () => {
            table.setAttribute('data-header-theme', colorConfig.theme || 'slate');
            getOrderedHeaders().forEach(th => {
                const c = th.getAttribute('data-col');
                const accent = colorConfig.accents && colorConfig.accents[c];
                if (accent) {
                    th.setAttribute('data-col-accent', accent);
                } else {
                    th.removeAttribute('data-col-accent');
                }
            });
            const pop = getPopoverForTable(tableId);
            if (pop) {
                pop.querySelectorAll('.theme-swatch-btn').forEach(btn => {
                    btn.classList.toggle('active', btn.getAttribute('data-theme') === (colorConfig.theme || 'slate'));
                });
            }
        };
        applyColors();

        // Apply Universal Column Widths via Dynamic <style>
        const applyWidths = () => {
            let wStyleEl = document.getElementById('radline-universal-widths-' + tableId);
            if (!wStyleEl) {
                wStyleEl = document.createElement('style');
                wStyleEl.id = 'radline-universal-widths-' + tableId;
                document.head.appendChild(wStyleEl);
            }
            const rules = [];
            Object.keys(colWidths).forEach(cKey => {
                const w = parseInt(colWidths[cKey], 10);
                if (w && w >= 40) {
                    rules.push(`
                        #${tableId} .th[data-col="${cKey}"],
                        #${tableId} .td[data-col="${cKey}"],
                        #${tableId} th[data-col="${cKey}"],
                        #${tableId} td[data-col="${cKey}"] {
                            width: ${w}px !important;
                            min-width: ${w}px !important;
                            max-width: ${w}px !important;
                            box-sizing: border-box !important;
                        }
                    `);
                }
            });
            wStyleEl.textContent = rules.join('\n');
        };
        applyWidths();

        let styleEl = document.getElementById('excel-collapsed-style-' + tableId);
        if (!styleEl) {
            styleEl = document.createElement('style');
            styleEl.id = 'excel-collapsed-style-' + tableId;
            document.head.appendChild(styleEl);
        }

        const groups = window.RADLINE_TABLE_GROUPS[tableId] || [];

        const syncPopoverGroupCheckboxes = () => {
            const pop = getPopoverForTable(tableId);
            if (!pop) return;
            pop.querySelectorAll('[data-target-col]').forEach(cb => {
                const c = cb.getAttribute('data-target-col');
                cb.checked = !collapsedCols.includes(c);
            });
            pop.querySelectorAll('.customize-group-cb').forEach(gcb => {
                const gId = gcb.getAttribute('data-group-id');
                const grp = groups.find(g => g.id === gId);
                if (!grp) return;
                const hiddenCount = grp.cols.filter(c => collapsedCols.includes(c)).length;
                gcb.checked = hiddenCount < grp.cols.length;
                gcb.indeterminate = hiddenCount > 0 && hiddenCount < grp.cols.length;
                const badge = gcb.closest('.customize-group-header')?.querySelector('.customize-group-badge');
                if (badge) {
                    badge.textContent = `${grp.cols.length - hiddenCount}/${grp.cols.length}`;
                }
            });
        };

        const updateTableCollapsedStyles = (persist = true) => {
            const allHeaders = getOrderedHeaders();
            const allColKeys = allHeaders.map(h => h.getAttribute('data-col'));

            const rules = [];
            collapsedCols.forEach(colKey => {
                rules.push(`
                    #${tableId} .th[data-col="${colKey}"],
                    #${tableId} .td[data-col="${colKey}"],
                    #${tableId} th[data-col="${colKey}"],
                    #${tableId} td[data-col="${colKey}"] {
                        display: none !important;
                    }
                `);
            });
            styleEl.textContent = rules.join('\n');

            if (persist) {
                window.saveRadlinePreference(storageKey, JSON.stringify(collapsedCols), collapsedCols.length === 0);
                try {
                    window.dispatchEvent(new CustomEvent('radline-excel-cols-updated', { detail: { tableId, collapsedCols } }));
                } catch (e) { }
            }
            syncPopoverGroupCheckboxes();

            // Render sleek unhide chip bar (Group Chips + Individual Column Chips)
            const targetContainer = table.closest('.table-container') || table;
            const parentCard = targetContainer.parentElement;
            let bar = parentCard ? parentCard.querySelector(`.collapsed-cols-bar[data-for="${tableId}"]`) : null;

            if (collapsedCols.length > 0) {
                if (!bar && parentCard) {
                    bar = document.createElement('div');
                    bar.className = 'collapsed-cols-bar';
                    bar.setAttribute('data-for', tableId);
                    parentCard.insertBefore(bar, targetContainer);
                }
                if (bar) {
                    const coveredByGroup = new Set();
                    const groupChipsHtml = [];

                    groups.forEach(grp => {
                        const presentCols = grp.cols.filter(c => allColKeys.includes(c));
                        if (presentCols.length >= 2 && presentCols.every(c => collapsedCols.includes(c))) {
                            presentCols.forEach(c => coveredByGroup.add(c));
                            const safeGrpLabel = escapeTableHtml(grp.label);
                            groupChipsHtml.push(`
                                <button type="button" class="collapsed-group-chip" data-group-id="${escapeTableHtml(grp.id)}" title="Click to unhide group '${safeGrpLabel}' (${presentCols.length} columns)">
                                    <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                                        <rect x="3" y="3" width="7" height="7"></rect>
                                        <rect x="14" y="3" width="7" height="7"></rect>
                                        <rect x="14" y="14" width="7" height="7"></rect>
                                        <rect x="3" y="14" width="7" height="7"></rect>
                                    </svg>
                                    <span>${safeGrpLabel} (${presentCols.length} cols)</span>
                                </button>
                            `);
                        }
                    });

                    const indivChipsHtml = collapsedCols
                        .filter(k => !coveredByGroup.has(k))
                        .map(k => {
                            const h = allHeaders.find(hdr => hdr.getAttribute('data-col') === k);
                            const name = h ? (h.getAttribute('data-original-title') || h.textContent.trim()) : k;
                            const safeName = escapeTableHtml(name);
                            const safeKey = escapeTableHtml(k);
                            return `
                                <button type="button" class="collapsed-col-chip" data-col="${safeKey}" title="Click to unhide column '${safeName}'">
                                    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                                        <line x1="12" y1="5" x2="12" y2="19"></line>
                                        <line x1="5" y1="12" x2="19" y2="12"></line>
                                    </svg>
                                    <span>${safeName}</span>
                                </button>
                            `;
                        });

                    bar.innerHTML = `
                        <div class="collapsed-cols-inner">
                            <div class="collapsed-cols-label">
                                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                    <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"></path>
                                    <line x1="1" y1="1" x2="23" y2="23"></line>
                                </svg>
                                <span>Hidden (${collapsedCols.length}):</span>
                            </div>
                            <div class="collapsed-cols-chips">${groupChipsHtml.concat(indivChipsHtml).join('')}</div>
                            <button type="button" class="collapsed-cols-unhide-all" title="Unhide all columns">Unhide all</button>
                        </div>
                    `;

                    bar.querySelectorAll('.collapsed-group-chip').forEach(gChip => {
                        gChip.onclick = (e) => {
                            e.stopPropagation();
                            const gId = gChip.getAttribute('data-group-id');
                            const grp = groups.find(g => g.id === gId);
                            if (grp) {
                                collapsedCols = collapsedCols.filter(c => !grp.cols.includes(c));
                                updateTableCollapsedStyles(true);
                            }
                        };
                    });

                    bar.querySelectorAll('.collapsed-col-chip').forEach(chip => {
                        chip.onclick = (e) => {
                            e.stopPropagation();
                            const col = chip.getAttribute('data-col');
                            collapsedCols = collapsedCols.filter(c => c !== col);
                            updateTableCollapsedStyles(true);
                        };
                    });

                    const unhideAllBtn = bar.querySelector('.collapsed-cols-unhide-all');
                    if (unhideAllBtn) {
                        unhideAllBtn.onclick = (e) => {
                            e.stopPropagation();
                            collapsedCols = [];
                            updateTableCollapsedStyles(true);
                        };
                    }
                }
            } else if (bar) {
                bar.remove();
            }

            // Remove existing divider handles
            table.querySelectorAll('.has-hidden-next, .has-hidden-prev').forEach(el => {
                el.classList.remove('has-hidden-next', 'has-hidden-prev');
                const h = el.querySelector('.excel-divider-handle');
                if (h) h.remove();
            });

            // Render visible expand [+] buttons on header divider borders
            const visibleHeaders = allHeaders.filter(h => !collapsedCols.includes(h.getAttribute('data-col')));

            if (visibleHeaders.length > 0) {
                const firstKey = visibleHeaders[0].getAttribute('data-col');
                const firstIdx = allColKeys.indexOf(firstKey);
                if (firstIdx > 0) {
                    const hiddenPreceding = allColKeys.slice(0, firstIdx).filter(k => collapsedCols.includes(k));
                    if (hiddenPreceding.length > 0) {
                        visibleHeaders[0].classList.add('has-hidden-prev');
                        table.querySelectorAll(`.td[data-col="${firstKey}"], td[data-col="${firstKey}"]`).forEach(td => td.classList.add('has-hidden-prev'));

                        const hiddenNames = escapeTableHtml(hiddenPreceding.map(k => {
                            const h = allHeaders.find(hdr => hdr.getAttribute('data-col') === k);
                            return h ? (h.getAttribute('data-original-title') || k) : k;
                        }).join(', '));

                        const handle = document.createElement('div');
                        handle.className = 'excel-divider-handle left-handle';
                        handle.title = `Click to unhide: ${hiddenNames}`;
                        handle.innerHTML = `
                            <button type="button" class="excel-unhide-btn" title="Click to unhide: ${hiddenNames}" aria-label="Unhide columns">
                                <svg width="8" height="8" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
                                    <line x1="12" y1="5" x2="12" y2="19"></line>
                                    <line x1="5" y1="12" x2="19" y2="12"></line>
                                </svg>
                            </button>
                        `;
                        handle.addEventListener('click', (ev) => {
                            ev.stopPropagation();
                            ev.preventDefault();
                            collapsedCols = collapsedCols.filter(k => !hiddenPreceding.includes(k));
                            updateTableCollapsedStyles(true);
                        });
                        visibleHeaders[0].appendChild(handle);
                    }
                }

                visibleHeaders.forEach((vh, i) => {
                    const curKey = vh.getAttribute('data-col');
                    const curIdx = allColKeys.indexOf(curKey);
                    const nextIdx = (i + 1 < visibleHeaders.length) ? allColKeys.indexOf(visibleHeaders[i + 1].getAttribute('data-col')) : allColKeys.length;

                    if (nextIdx > curIdx + 1) {
                        const hiddenBetween = allColKeys.slice(curIdx + 1, nextIdx).filter(k => collapsedCols.includes(k));
                        if (hiddenBetween.length > 0) {
                            vh.classList.add('has-hidden-next');
                            table.querySelectorAll(`.td[data-col="${curKey}"], td[data-col="${curKey}"]`).forEach(td => td.classList.add('has-hidden-next'));

                            const hiddenNames = escapeTableHtml(hiddenBetween.map(k => {
                                const h = allHeaders.find(hdr => hdr.getAttribute('data-col') === k);
                                return h ? (h.getAttribute('data-original-title') || k) : k;
                            }).join(', '));

                            const handle = document.createElement('div');
                            handle.className = 'excel-divider-handle right-handle';
                            handle.title = `Click to unhide: ${hiddenNames}`;
                            handle.innerHTML = `
                                <button type="button" class="excel-unhide-btn" title="Click to unhide: ${hiddenNames}" aria-label="Unhide columns">
                                    <svg width="8" height="8" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
                                        <line x1="12" y1="5" x2="12" y2="19"></line>
                                        <line x1="5" y1="12" x2="19" y2="12"></line>
                                    </svg>
                                </button>
                            `;
                            handle.addEventListener('click', (ev) => {
                                ev.stopPropagation();
                                ev.preventDefault();
                                collapsedCols = collapsedCols.filter(k => !hiddenBetween.includes(k));
                                updateTableCollapsedStyles(true);
                            });
                            vh.appendChild(handle);
                        }
                    }
                });
            }
        };

        const moveColumn = (colKey, delta) => {
            const idx = colOrder.indexOf(colKey);
            if (idx === -1) return;
            const targetIdx = idx + delta;
            if (targetIdx < 0 || targetIdx >= colOrder.length) return;
            const copy = [...colOrder];
            const [removed] = copy.splice(idx, 1);
            copy.splice(targetIdx, 0, removed);
            colOrder = copy;
            applyColumnOrderToDOM(table, colOrder);
            window.saveRadlinePreference(orderKey, JSON.stringify(colOrder), false);
            updateTableCollapsedStyles(false);
        };

        // Upgrade Customize View Popover (Theme Picker + Grouped Checkboxes + Reorder Controls)
        const popover = getPopoverForTable(tableId);
        if (popover && !popover.hasAttribute('data-radline-enhanced')) {
            popover.setAttribute('data-radline-enhanced', 'true');

            const resetBtn = popover.querySelector('button[onclick*="reset"]');
            if (resetBtn) {
                resetBtn.addEventListener('click', () => {
                    window.resetRadlineTableCustomization(tableId);
                });
            }

            const firstSectionTitle = popover.querySelector('.customize-section-title');
            if (firstSectionTitle) {
                const themeTitle = document.createElement('div');
                themeTitle.className = 'customize-section-title';
                themeTitle.textContent = 'Header Color Theme';

                const swatchesWrap = document.createElement('div');
                swatchesWrap.className = 'customize-theme-swatches';
                swatchesWrap.innerHTML = window.RADLINE_HEADER_THEMES.map(t => `
                    <button type="button" class="theme-swatch-btn ${colorConfig.theme === t.id ? 'active' : ''}" data-theme="${t.id}" title="${t.label} Header Theme">
                        <span class="theme-swatch-dot" style="background: ${t.dot};"></span>
                        <span>${t.label}</span>
                    </button>
                `).join('');

                swatchesWrap.querySelectorAll('.theme-swatch-btn').forEach(btn => {
                    btn.onclick = (ev) => {
                        ev.stopPropagation();
                        colorConfig.theme = btn.getAttribute('data-theme') || 'slate';
                        applyColors();
                        window.saveRadlinePreference(colorKey, JSON.stringify(colorConfig), false);
                    };
                });

                firstSectionTitle.parentNode.insertBefore(themeTitle, firstSectionTitle);
                firstSectionTitle.parentNode.insertBefore(swatchesWrap, firstSectionTitle);
            }

            const colCheckboxes = Array.from(popover.querySelectorAll('input[data-target-col]'));
            const firstColGrid = colCheckboxes[0] ? colCheckboxes[0].closest('.customize-checkbox-grid') : null;
            if (firstColGrid && groups.length > 0) {
                if (firstSectionTitle) {
                    firstSectionTitle.textContent = 'Column Groups & Order (Drag headers or use ← →)';
                }
                const labelMap = {};
                colCheckboxes.forEach(cb => {
                    const cKey = cb.getAttribute('data-target-col');
                    const lbl = cb.closest('label');
                    if (cKey && lbl) labelMap[cKey] = lbl;
                });

                const groupsContainer = document.createElement('div');
                groups.forEach(grp => {
                    const grpLabels = grp.cols.map(c => labelMap[c]).filter(Boolean);
                    if (grpLabels.length === 0) return;

                    const block = document.createElement('div');
                    block.className = 'customize-group-block';
                    block.innerHTML = `
                        <div class="customize-group-header">
                            <label>
                                <input type="checkbox" class="customize-group-cb" data-group-id="${escapeTableHtml(grp.id)}">
                                <span>${escapeTableHtml(grp.label)}</span>
                            </label>
                            <span class="customize-group-badge">${grpLabels.length}/${grpLabels.length}</span>
                        </div>
                        <div class="customize-group-cols"></div>
                    `;
                    const colsBox = block.querySelector('.customize-group-cols');
                    grpLabels.forEach(lbl => {
                        const cb = lbl.querySelector('input[data-target-col]');
                        const cKey = cb.getAttribute('data-target-col');
                        const rowWrap = document.createElement('div');
                        rowWrap.className = 'customize-col-row';
                        rowWrap.setAttribute('data-col-row', cKey);

                        const reorderBtns = document.createElement('div');
                        reorderBtns.className = 'customize-col-reorder-btns';
                        reorderBtns.innerHTML = `
                            <button type="button" class="btn-col-move" data-dir="-1" title="Move column left">&larr;</button>
                            <button type="button" class="btn-col-move" data-dir="1" title="Move column right">&rarr;</button>
                        `;
                        reorderBtns.querySelectorAll('.btn-col-move').forEach(mBtn => {
                            mBtn.onclick = (ev) => {
                                ev.stopPropagation();
                                ev.preventDefault();
                                const dir = parseInt(mBtn.getAttribute('data-dir'), 10);
                                moveColumn(cKey, dir);
                            };
                        });

                        rowWrap.appendChild(lbl);
                        rowWrap.appendChild(reorderBtns);
                        colsBox.appendChild(rowWrap);
                    });

                    const grpCb = block.querySelector('.customize-group-cb');
                    grpCb.onchange = (ev) => {
                        ev.stopPropagation();
                        const shouldShow = grpCb.checked;
                        grp.cols.forEach(c => {
                            const cb = popover.querySelector(`input[data-target-col="${c}"]`);
                            if (cb) cb.checked = shouldShow;
                            if (shouldShow) {
                                collapsedCols = collapsedCols.filter(x => x !== c);
                            } else if (!collapsedCols.includes(c)) {
                                collapsedCols.push(c);
                            }
                        });
                        updateTableCollapsedStyles(true);
                        const sampleCb = popover.querySelector('input[data-target-col]');
                        if (sampleCb && typeof sampleCb.onchange === 'function') {
                            sampleCb.onchange();
                        }
                    };

                    groupsContainer.appendChild(block);
                });

                firstColGrid.parentNode.replaceChild(groupsContainer, firstColGrid);
            }
        }

        // Setup each header: title, collapse button, resizer handle, drag-and-drop reorder
        getOrderedHeaders().forEach((th) => {
            const colKey = th.getAttribute('data-col');
            if (colKey === 'actions' || colKey === 'select' || colKey === 'row_select') return;

            const titleText = th.getAttribute('data-original-title') || th.textContent.trim();
            th.setAttribute('data-original-title', titleText);
            const safeTitleText = escapeTableHtml(titleText);

            let thContent = th.querySelector('.th-content');
            if (!thContent) {
                th.innerHTML = `
                    <div class="th-content">
                        <span class="th-title" title="${safeTitleText} (Drag to reorder — Right-click for options)">${safeTitleText}</span>
                        <button type="button" class="btn-th-collapse" title="Collapse column '${safeTitleText}'" aria-label="Collapse column">
                            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                                <line x1="5" y1="12" x2="19" y2="12"></line>
                            </svg>
                        </button>
                    </div>
                `;
            } else if (!thContent.querySelector('.btn-th-collapse')) {
                const collapseBtn = document.createElement('button');
                collapseBtn.type = 'button';
                collapseBtn.className = 'btn-th-collapse';
                collapseBtn.title = `Collapse column '${titleText}'`;
                collapseBtn.setAttribute('aria-label', 'Collapse column');
                collapseBtn.innerHTML = `
                    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                        <line x1="5" y1="12" x2="19" y2="12"></line>
                    </svg>
                `;
                thContent.appendChild(collapseBtn);
            }

            const collapseBtn = th.querySelector('.btn-th-collapse');
            if (collapseBtn) {
                collapseBtn.onclick = (e) => {
                    e.stopPropagation();
                    e.preventDefault();
                    if (!collapsedCols.includes(colKey)) {
                        collapsedCols.push(colKey);
                        updateTableCollapsedStyles(true);
                    }
                };
            }

            // Inject Column Resizer Handle
            let resizer = th.querySelector('.table-col-resizer');
            if (!resizer) {
                resizer = document.createElement('div');
                resizer.className = 'table-col-resizer';
                resizer.title = 'Drag to resize column — Double-click to reset width';
                th.appendChild(resizer);
            }

            resizer.onmousedown = (e) => {
                e.stopPropagation();
                e.preventDefault();
                th.setAttribute('draggable', 'false');
                const startX = e.clientX;
                const startWidth = th.getBoundingClientRect().width;
                resizer.classList.add('is-resizing');
                document.body.style.cursor = 'col-resize';
                document.body.style.userSelect = 'none';

                const onMouseMove = (moveEv) => {
                    const newWidth = Math.max(50, Math.round(startWidth + (moveEv.clientX - startX)));
                    colWidths[colKey] = newWidth;
                    applyWidths();
                };

                const onMouseUp = () => {
                    resizer.classList.remove('is-resizing');
                    document.body.style.cursor = '';
                    document.body.style.userSelect = '';
                    th.setAttribute('draggable', 'true');
                    window.saveRadlinePreference(widthKey, JSON.stringify(colWidths), false);
                    document.removeEventListener('mousemove', onMouseMove);
                    document.removeEventListener('mouseup', onMouseUp);
                };

                document.addEventListener('mousemove', onMouseMove);
                document.addEventListener('mouseup', onMouseUp);
            };

            resizer.ondblclick = (e) => {
                e.stopPropagation();
                e.preventDefault();
                delete colWidths[colKey];
                applyWidths();
                window.saveRadlinePreference(widthKey, JSON.stringify(colWidths), Object.keys(colWidths).length === 0);
            };

            // Enable Column Context Menu on Header
            th.oncontextmenu = (e) => {
                handleRadlineHeaderContextMenu(e, table, th);
            };

            // Enable HTML5 Drag & Drop Column Reordering
            th.setAttribute('draggable', 'true');
            th.ondragstart = (e) => {
                if (resizer.classList.contains('is-resizing')) {
                    e.preventDefault();
                    return;
                }
                th.classList.add('is-col-dragging');
                e.dataTransfer.effectAllowed = 'move';
                e.dataTransfer.setData('text/plain', colKey);
                table._draggingColKey = colKey;
            };

            th.ondragover = (e) => {
                const draggedCol = table._draggingColKey;
                if (!draggedCol || draggedCol === colKey) return;
                e.preventDefault();
                e.dataTransfer.dropEffect = 'move';
                const rect = th.getBoundingClientRect();
                const isLeft = (e.clientX - rect.left) < rect.width / 2;
                th.classList.toggle('col-drop-left', isLeft);
                th.classList.toggle('col-drop-right', !isLeft);
            };

            th.ondragleave = () => {
                th.classList.remove('col-drop-left', 'col-drop-right');
            };

            th.ondrop = (e) => {
                e.preventDefault();
                const isLeft = th.classList.contains('col-drop-left');
                th.classList.remove('col-drop-left', 'col-drop-right');
                const draggedCol = table._draggingColKey || e.dataTransfer.getData('text/plain');
                if (!draggedCol || draggedCol === colKey) return;

                const nextOrder = colOrder.filter(c => c !== draggedCol);
                const targetIdx = nextOrder.indexOf(colKey);
                if (targetIdx !== -1) {
                    nextOrder.splice(isLeft ? targetIdx : targetIdx + 1, 0, draggedCol);
                    colOrder = nextOrder;
                    applyColumnOrderToDOM(table, colOrder);
                    window.saveRadlinePreference(orderKey, JSON.stringify(colOrder), false);
                    updateTableCollapsedStyles(false);
                }
            };

            th.ondragend = () => {
                table._draggingColKey = null;
                getOrderedHeaders().forEach(h => h.classList.remove('is-col-dragging', 'col-drop-left', 'col-drop-right'));
            };
        });

        updateTableCollapsedStyles(false);
    });
};

// Context menu for Column Colors, Grouping, Reordering & Hiding
function showColumnContextMenu(opts) {
    const { e, colKey, titleText, collapsedCols, colorConfig, groups, onUpdateCollapsed, onUpdateColors, onMoveCol, onResetAll } = opts;
    const safeTitleText = escapeTableHtml(titleText);
    let menu = document.getElementById('table-col-context-menu');
    if (!menu) {
        menu = document.createElement('div');
        menu.id = 'table-col-context-menu';
        menu.className = 'table-col-context-menu';
        document.body.appendChild(menu);
    }

    const hasHidden = collapsedCols && collapsedCols.length > 0;
    const curAccent = (colorConfig.accents && colorConfig.accents[colKey]) || '';
    const curTheme = colorConfig.theme || 'slate';
    const colGroup = (groups || []).find(g => g.cols.includes(colKey));
    const isGroupHidden = colGroup ? colGroup.cols.every(c => collapsedCols.includes(c)) : false;

    const accentSwatchesHtml = window.RADLINE_COL_ACCENTS.map(a => `
        <button type="button" class="col-swatch-circle ${curAccent === a.id ? 'active' : ''}" data-accent="${a.id}" title="Column Highlight: ${a.label}" style="background: ${a.dot};"></button>
    `).join('');

    const themeSwatchesHtml = window.RADLINE_HEADER_THEMES.map(t => `
        <button type="button" class="col-swatch-circle ${curTheme === t.id ? 'active' : ''}" data-theme="${t.id}" title="Table Theme: ${t.label}" style="background: ${t.dot};"></button>
    `).join('');

    menu.innerHTML = `
        <div class="col-menu-label">Column "${safeTitleText}" Highlight</div>
        <div class="col-menu-swatches">${accentSwatchesHtml}</div>

        <div class="col-menu-label">Table Header Theme</div>
        <div class="col-menu-swatches">${themeSwatchesHtml}</div>

        <div class="col-menu-divider"></div>
        <button type="button" class="col-menu-item" id="col-menu-move-left">
            <span>&larr; Move Column Left</span>
        </button>
        <button type="button" class="col-menu-item" id="col-menu-move-right">
            <span>&rarr; Move Column Right</span>
        </button>

        <div class="col-menu-divider"></div>
        <button type="button" class="col-menu-item" id="col-menu-hide">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <line x1="5" y1="12" x2="19" y2="12"></line>
            </svg>
            <span>Collapse column "${safeTitleText}"</span>
        </button>
        ${colGroup ? `
        <button type="button" class="col-menu-item" id="col-menu-toggle-group">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect x="3" y="3" width="7" height="7"></rect>
                <rect x="14" y="3" width="7" height="7"></rect>
                <rect x="14" y="14" width="7" height="7"></rect>
                <rect x="3" y="14" width="7" height="7"></rect>
            </svg>
            <span>${isGroupHidden ? 'Unhide' : 'Hide'} Group "${escapeTableHtml(colGroup.label)}"</span>
        </button>
        ` : ''}
        ${hasHidden ? `
        <button type="button" class="col-menu-item" id="col-menu-unhide-all">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path>
                <circle cx="12" cy="12" r="3"></circle>
            </svg>
            <span>Unhide all columns (${collapsedCols.length})</span>
        </button>
        ` : ''}
        <div class="col-menu-divider"></div>
        <button type="button" class="col-menu-item" id="col-menu-reset-all" style="color: #64748b;">
            <span>Reset Default View</span>
        </button>
    `;

    menu.classList.add('is-active');
    menu.style.setProperty('display', 'block', 'important');
    menu.style.setProperty('z-index', '9999999', 'important');
    const menuWidth = 240;
    const menuHeight = 310;
    let left = e.clientX;
    let top = e.clientY;
    if (left + menuWidth > window.innerWidth) left = Math.max(8, window.innerWidth - menuWidth - 12);
    if (top + menuHeight > window.innerHeight) top = Math.max(8, window.innerHeight - menuHeight - 12);
    menu.style.left = left + 'px';
    menu.style.top = top + 'px';

    menu.querySelectorAll('[data-accent]').forEach(btn => {
        btn.onclick = (ev) => {
            ev.stopPropagation();
            const acc = btn.getAttribute('data-accent');
            const nextAccents = Object.assign({}, colorConfig.accents || {});
            if (acc) {
                nextAccents[colKey] = acc;
            } else {
                delete nextAccents[colKey];
            }
            onUpdateColors({ theme: colorConfig.theme || 'slate', accents: nextAccents });
            menu.style.display = 'none';
        };
    });

    menu.querySelectorAll('[data-theme]').forEach(btn => {
        btn.onclick = (ev) => {
            ev.stopPropagation();
            const thm = btn.getAttribute('data-theme') || 'slate';
            onUpdateColors({ theme: thm, accents: colorConfig.accents || {} });
            menu.style.display = 'none';
        };
    });

    const moveLeftBtn = menu.querySelector('#col-menu-move-left');
    if (moveLeftBtn) {
        moveLeftBtn.onclick = (ev) => {
            ev.stopPropagation();
            menu.style.display = 'none';
            onMoveCol(-1);
        };
    }
    const moveRightBtn = menu.querySelector('#col-menu-move-right');
    if (moveRightBtn) {
        moveRightBtn.onclick = (ev) => {
            ev.stopPropagation();
            menu.style.display = 'none';
            onMoveCol(1);
        };
    }

    const hideBtn = menu.querySelector('#col-menu-hide');
    if (hideBtn) {
        hideBtn.onclick = (ev) => {
            ev.stopPropagation();
            menu.style.display = 'none';
            if (!collapsedCols.includes(colKey)) {
                onUpdateCollapsed([...collapsedCols, colKey]);
            }
        };
    }

    const toggleGrpBtn = menu.querySelector('#col-menu-toggle-group');
    if (toggleGrpBtn && colGroup) {
        toggleGrpBtn.onclick = (ev) => {
            ev.stopPropagation();
            menu.style.display = 'none';
            if (isGroupHidden) {
                onUpdateCollapsed(collapsedCols.filter(c => !colGroup.cols.includes(c)));
            } else {
                const merged = Array.from(new Set([...collapsedCols, ...colGroup.cols]));
                onUpdateCollapsed(merged);
            }
        };
    }

    const unhideAllBtn = menu.querySelector('#col-menu-unhide-all');
    if (unhideAllBtn) {
        unhideAllBtn.onclick = (ev) => {
            ev.stopPropagation();
            menu.style.display = 'none';
            onUpdateCollapsed([]);
        };
    }

    const resetAllBtn = menu.querySelector('#col-menu-reset-all');
    if (resetAllBtn) {
        resetAllBtn.onclick = (ev) => {
            ev.stopPropagation();
            menu.style.display = 'none';
            onResetAll();
        };
    }
}

// Global Event Delegation for Right-Click (Context Menu) on Table Headers
function handleRadlineHeaderContextMenu(e, table, th) {
    if (!table || !th) return;
    const colKey = th.getAttribute('data-col');
    if (!colKey || colKey === 'actions' || colKey === 'select' || colKey === 'row_select') return;

    e.preventDefault();
    e.stopPropagation();

    const tableId = table.id || table.getAttribute('data-table-id') ||
        (table.classList.contains('sales-log-table') ? 'sales-log-table' :
            (table.classList.contains('inventory-stock-table') ? 'inventory-stock-table' :
                (table.classList.contains('monthly-inventory-table') ? 'monthly-inventory-table' :
                    (table.classList.contains('items-table') ? 'items-table' :
                        (table.classList.contains('receiving-logs-table') ? 'receiving-logs-table' :
                            (table.classList.contains('adjustment-logs-table') ? 'adjustment-logs-table' :
                                'sales-log-table'))))));
    if (!table.id) table.id = tableId;

    if (!table._defaultColOrder) {
        table._defaultColOrder = Array.from(table.querySelectorAll('.thead .th, thead th'))
            .map(h => h.getAttribute('data-col'))
            .filter(c => c && c !== 'actions' && c !== 'select' && c !== 'row_select');
    }

    const storageKey = 'radline_collapsed_cols_' + tableId;
    const colorKey = 'radline_header_colors_' + tableId;
    const orderKey = 'radline_col_order_' + tableId;

    let collapsedCols = [];
    try {
        collapsedCols = JSON.parse(localStorage.getItem(storageKey) || '[]');
    } catch (err) { }

    let colorConfig = { theme: 'slate', accents: {} };
    try {
        const parsedColors = JSON.parse(localStorage.getItem(colorKey) || 'null');
        if (parsedColors && typeof parsedColors === 'object') {
            colorConfig.theme = parsedColors.theme || 'slate';
            colorConfig.accents = parsedColors.accents || {};
        }
    } catch (err) { }

    const groups = window.RADLINE_TABLE_GROUPS[tableId] || [];
    const titleText = th.getAttribute('data-original-title') || th.querySelector('.th-title')?.textContent.trim() || th.textContent.trim();

    showColumnContextMenu({
        e,
        tableId,
        colKey,
        titleText,
        collapsedCols,
        colorConfig,
        groups,
        onUpdateCollapsed: (newCols) => {
            try {
                localStorage.setItem(storageKey, JSON.stringify(newCols));
                window.saveRadlinePreference(storageKey, JSON.stringify(newCols), newCols.length === 0);
            } catch (err) { }
            if (typeof window.initExcelCollapsibleColumns === 'function') {
                window.initExcelCollapsibleColumns();
            }
        },
        onUpdateColors: (newColors) => {
            colorConfig = newColors;
            try {
                localStorage.setItem(colorKey, JSON.stringify(colorConfig));
                window.saveRadlinePreference(colorKey, JSON.stringify(colorConfig), false);
            } catch (err) { }
            if (typeof window.initExcelCollapsibleColumns === 'function') {
                window.initExcelCollapsibleColumns();
            }
        },
        onMoveCol: (delta) => {
            let colOrder = [...(table._defaultColOrder || [])];
            try {
                const savedOrder = JSON.parse(localStorage.getItem(orderKey) || 'null');
                if (Array.isArray(savedOrder) && savedOrder.length > 0) {
                    colOrder = savedOrder;
                }
            } catch (err) { }
            const idx = colOrder.indexOf(colKey);
            if (idx === -1) return;
            const targetIdx = idx + delta;
            if (targetIdx < 0 || targetIdx >= colOrder.length) return;
            const copy = [...colOrder];
            const [removed] = copy.splice(idx, 1);
            copy.splice(targetIdx, 0, removed);
            colOrder = copy;
            applyColumnOrderToDOM(table, colOrder);
            window.saveRadlinePreference(orderKey, JSON.stringify(colOrder), false);
            if (typeof window.initExcelCollapsibleColumns === 'function') {
                window.initExcelCollapsibleColumns();
            }
        },
        onResetAll: () => {
            if (typeof window.resetRadlineTableCustomization === 'function') {
                window.resetRadlineTableCustomization(tableId);
            }
        }
    });
}

document.addEventListener('contextmenu', function (e) {
    const th = e.target.closest('.thead .th, thead th, .th[data-col], th[data-col]');
    if (!th) {
        const menu = document.getElementById('table-col-context-menu');
        if (menu && !menu.contains(e.target)) {
            menu.classList.remove('is-active');
            menu.style.display = 'none';
        }
        return;
    }

    const table = th.closest('.table, .resizable-table, .sales-log-table, table.table-scroll-md, table.table-scroll-lg, .excel-table, table');
    if (!table) return;

    handleRadlineHeaderContextMenu(e, table, th);
});

// Global dismiss for context menu on click outside
document.addEventListener('click', function (e) {
    const menu = document.getElementById('table-col-context-menu');
    if (menu && !menu.contains(e.target)) {
        menu.classList.remove('is-active');
        menu.style.display = 'none';
    }
});

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', window.initExcelCollapsibleColumns);
} else {
    window.initExcelCollapsibleColumns();
}
document.body.addEventListener('htmx:afterSwap', function () {
    window.initExcelCollapsibleColumns();
});

// PWA Service Worker Registration
if ('serviceWorker' in navigator) {
    window.addEventListener('load', function () {
        navigator.serviceWorker.register('/sw.js', { scope: '/' })
            .catch(function (err) {
                console.warn('PWA ServiceWorker registration failed:', err);
            });
    });
}

// Online / Offline Status Toasts
window.addEventListener('offline', function () {
    document.body.dispatchEvent(new CustomEvent('show-toast', {
        detail: { type: 'warning', message: 'Network offline. Using cached shell.' }
    }));
});

window.addEventListener('online', function () {
    document.body.dispatchEvent(new CustomEvent('show-toast', {
        detail: { type: 'success', message: 'Connection restored. Back online.' }
    }));
});
