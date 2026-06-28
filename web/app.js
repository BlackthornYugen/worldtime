// Timezone abbreviation mappings (matching Go tzdb)
const AbbreviationMap = {
    "utc": "UTC",
    "gmt": "Etc/GMT",
    "est": "America/New_York",
    "edt": "America/New_York",
    "et": "America/New_York",
    "cst": "America/Chicago",
    "cdt": "America/Chicago",
    "ct": "America/Chicago",
    "mst": "America/Denver",
    "mdt": "America/Denver",
    "mt": "America/Denver",
    "pst": "America/Los_Angeles",
    "pdt": "America/Los_Angeles",
    "pt": "America/Los_Angeles",
    "akst": "America/Anchorage",
    "akdt": "America/Anchorage",
    "hst": "Pacific/Honolulu",
    "ast": "America/Halifax",
    "adt": "America/Halifax",
    "bst": "Europe/London",
    "cet": "Europe/Paris",
    "cest": "Europe/Paris",
    "eet": "Europe/Athens",
    "eest": "Europe/Athens",
    "msk": "Europe/Moscow",
    "ist": "Asia/Kolkata",
    "jst": "Asia/Tokyo",
    "kst": "Asia/Seoul",
    "aest": "Australia/Sydney",
    "aedt": "Australia/Sydney",
    "aet": "Australia/Sydney",
    "awst": "Australia/Perth",
    "acst": "Australia/Adelaide",
    "acdt": "Australia/Adelaide",
    "nzst": "Pacific/Auckland",
    "nzdt": "Pacific/Auckland",
    "nzt": "Pacific/Auckland",
    "sgt": "Asia/Singapore",
    "hkt": "Asia/Hong_Kong",
    "wet": "Europe/Lisbon",
    "west": "Europe/Lisbon",
    "cat": "Africa/Maputo",
    "eat": "Africa/Nairobi",
    "wat": "Africa/Lagos",
};

// Autocomplete searches are queried dynamically via the backend API /api/search


// App state
let timezones = []; // Array of objects: { tz: string, friendlyName: string }
let focusTz = null;
let selectedHour = new Date().getHours(); // 0-23
let selectedDate = new Date(); // Date object
let is12hFormat = false;
let dragSrcEl = null;

// DOM Elements
const tzSearch = document.getElementById("tz-search");
const clearSearch = document.getElementById("clear-search");
const autocompleteList = document.getElementById("autocomplete-list");
const dateSelect = document.getElementById("date-select");
const format12hBtn = document.getElementById("format-12h");
const format24hBtn = document.getElementById("format-24h");
const sortBtn = document.getElementById("sort-btn");
const shareBtn = document.getElementById("share-btn");
const tzListContainer = document.getElementById("tz-list");

// Init application
function init() {
    const urlParams = new URLSearchParams(window.location.search);
    focusTz = urlParams.get("focus");

    // Set date input to today
    const yyyy = selectedDate.getFullYear();
    const mm = String(selectedDate.getMonth() + 1).padStart(2, '0');
    const dd = String(selectedDate.getDate()).padStart(2, '0');
    dateSelect.value = `${yyyy}-${mm}-${dd}`;

    // Set up event listeners
    setupEventListeners();

    // Check URL query parameters (legacy fallback) or clean path
    const path = window.location.pathname;
    if (path && path !== "/") {
        fetch(`/api/resolve?path=${encodeURIComponent(path)}`)
            .then(res => res.json())
            .then(data => {
                if (data && data.length > 0) {
                    timezones = data;
                } else {
                    loadDefaultTimezones();
                }
                finishInit();
            })
            .catch(err => {
                console.error("Error resolving path timezones:", err);
                loadDefaultTimezones();
                finishInit();
            });
    } else {
        const queryZones = parseQueryParams();
        if (queryZones.length > 0) {
            timezones = queryZones;
            updateUrl(); // upgrade to clean path
        } else {
            loadDefaultTimezones();
            updateUrl();
        }
        finishInit();
    }
}

function loadDefaultTimezones() {
    const localZone = Intl.DateTimeFormat().resolvedOptions().timeZone;
    timezones = [
        { tz: localZone, friendlyName: "Local", searchTerm: "Local" },
        { tz: "UTC", friendlyName: "UTC", searchTerm: "UTC" },
        { tz: "America/New_York", friendlyName: "New York", searchTerm: "America/New_York" },
        { tz: "Europe/Paris", friendlyName: "Paris", searchTerm: "Europe/Paris" },
        { tz: "Asia/Tokyo", friendlyName: "Tokyo", searchTerm: "Asia/Tokyo" }
    ];
}

function finishInit() {
    if (timezones.length > 0) {
        const parts = getTzParts(new Date(), timezones[0].tz);
        selectedHour = parts.hour;
    }
    render();
}

// Check if timezone string is valid
function isValidTimeZone(tz) {
    try {
        Intl.DateTimeFormat(undefined, { timeZone: tz });
        return true;
    } catch (e) {
        return false;
    }
}

// Normalize capitalization (e.g. "america/new_york" -> "America/New_York")
function normalizeTzName(str) {
    return str.split('/').map(segment => {
        return segment.split('_').map(word => {
            if (word.length === 0) return '';
            return word[0].toUpperCase() + word.slice(1).toLowerCase();
        }).join('_');
    }).join('/');
}

// Parse query params (e.g. legacy ?tz=America/New_York&friendlyName=Waterloo)
function parseQueryParams() {
    const params = new URLSearchParams(window.location.search);
    const tzs = params.getAll("tz");
    const friendlyNames = params.getAll("friendlyName");

    const list = [];
    for (let i = 0; i < tzs.length; i++) {
        const tz = tzs[i];
        if (isValidTimeZone(tz)) {
            const friendlyName = friendlyNames[i] || getFriendlyName(tz);
            list.push({ tz, friendlyName, searchTerm: tz });
        }
    }
    return list;
}

// Update address bar using clean paths
function updateUrl() {
    if (timezones.length === 0) {
        history.replaceState(null, "", "/");
        return;
    }

    const segments = timezones.map(item => {
        let term = item.searchTerm || item.tz;
        let res = term.replace(/\+/g, '%2B').replace(/ /g, '+');
        
        if (item.friendlyName && item.friendlyName !== term && item.friendlyName !== getFriendlyName(item.tz)) {
            let fn = item.friendlyName.replace(/\+/g, '%2B').replace(/ /g, '+');
            res = `${res}+as+${fn}`;
        }
        return res;
    });
    
    // update URL
    const newPath = "/" + segments.join("/");
    let searchParams = "";
    if (focusTz) {
        searchParams = "?focus=" + encodeURIComponent(focusTz);
    }
    history.replaceState(null, "", newPath + searchParams);
}

// Format date into specific parts for timezone math
function getTzParts(date, timeZone) {
    const formatter = new Intl.DateTimeFormat('en-US', {
        timeZone,
        hour12: false,
        year: 'numeric',
        month: 'numeric',
        day: 'numeric',
        hour: 'numeric',
        minute: 'numeric',
        second: 'numeric'
    });
    const parts = formatter.formatToParts(date);
    const res = {};
    for (const p of parts) {
        res[p.type] = p.value;
    }
    return {
        year: parseInt(res.year),
        month: parseInt(res.month),
        day: parseInt(res.day),
        hour: parseInt(res.hour) === 24 ? 0 : parseInt(res.hour),
        minute: parseInt(res.minute)
    };
}

// Get standard date object at specific hour in target timezone
function getDateTimeInZone(dateObj, hour, timeZone) {
    const yyyy = dateObj.getFullYear();
    const mm = dateObj.getMonth();
    const dd = dateObj.getDate();
    let utcTime = Date.UTC(yyyy, mm, dd, hour, 0, 0, 0);

    for (let i = 0; i < 3; i++) {
        const parts = getTzParts(new Date(utcTime), timeZone);
        const diffHours = (hour - parts.hour) + (dd - parts.day) * 24;
        if (diffHours === 0) break;
        utcTime += diffHours * 60 * 60 * 1000;
    }
    return new Date(utcTime);
}

// Get raw offset in minutes for a timezone at a specific date
function getRawOffsetMinutes(date, timeZone) {
    const formatter = new Intl.DateTimeFormat('en-US', {
        timeZone,
        timeZoneName: 'longOffset'
    });
    const parts = formatter.formatToParts(date);
    const offsetPart = parts.find(p => p.type === 'timeZoneName');
    if (!offsetPart) return 0;

    const val = offsetPart.value; // e.g. "GMT-05:00", "GMT+05:30", "GMT"
    if (val === 'GMT') return 0;

    const match = val.match(/GMT([+-])(\d+):(\d+)/);
    if (!match) return 0;

    const sign = match[1] === '-' ? -1 : 1;
    const hours = parseInt(match[2], 10);
    const minutes = parseInt(match[3], 10);
    return sign * (hours * 60 + minutes);
}

// Get relative offset string from a base timezone (e.g. +0, +9.5, -6)
function getRelativeOffsetStr(date, timeZone, baseTimeZone) {
    const baseMinutes = getRawOffsetMinutes(date, baseTimeZone);
    const zoneMinutes = getRawOffsetMinutes(date, timeZone);
    const diffMinutes = zoneMinutes - baseMinutes;
    const diffHours = diffMinutes / 60;

    if (diffHours === 0) {
        return "+0";
    }
    const sign = diffHours > 0 ? "+" : "";
    if (diffHours === Math.round(diffHours)) {
        return `${sign}${diffHours}`;
    } else {
        return `${sign}${diffHours.toFixed(1)}`;
    }
}

// Get offset label relative to UTC (e.g. UTC-5, UTC+5.5)
function getOffsetStr(date, timeZone) {
    const formatter = new Intl.DateTimeFormat('en-US', {
        timeZone,
        timeZoneName: 'longOffset'
    });
    const parts = formatter.formatToParts(date);
    const offsetPart = parts.find(p => p.type === 'timeZoneName');
    if (!offsetPart) return 'UTC';

    let val = offsetPart.value.replace('GMT', 'UTC');
    if (val === 'UTC') return 'UTC';

    val = val.replace(/([+-])0(\d)/, '$1$2'); // remove leading zero
    val = val.replace(':00', ''); // remove trailing :00
    return val;
}

// Get friendly timezone name (e.g., "America/New_York" -> "New York")
function getFriendlyName(name) {
    if (name === "UTC" || name === "GMT") return name;
    const parts = name.split('/');
    return parts[parts.length - 1].replace(/_/g, ' ');
}

// Set up UI Event listeners
function setupEventListeners() {
    // Search inputs
    tzSearch.addEventListener("input", handleSearchInput);
    tzSearch.addEventListener("focus", handleSearchInput);
    clearSearch.addEventListener("click", () => {
        tzSearch.value = "";
        clearSearch.classList.add("hide");
        autocompleteList.classList.add("hide");
    });

    // Close dropdown on click outside
    document.addEventListener("click", (e) => {
        if (!tzSearch.contains(e.target) && !autocompleteList.contains(e.target)) {
            autocompleteList.classList.add("hide");
        }
    });

    // Date select
    dateSelect.addEventListener("change", (e) => {
        if (e.target.value) {
            const [y, m, d] = e.target.value.split('-').map(Number);
            selectedDate = new Date(y, m - 1, d);
            render();
        }
    });

    // Format toggle
    format12hBtn.addEventListener("click", () => {
        is12hFormat = true;
        format12hBtn.classList.add("active");
        format24hBtn.classList.remove("active");
        render();
    });
    format24hBtn.addEventListener("click", () => {
        is12hFormat = false;
        format24hBtn.classList.add("active");
        format12hBtn.classList.remove("active");
        render();
    });

    // Share link button
    shareBtn.addEventListener("click", () => {
        const url = window.location.href;
        navigator.clipboard.writeText(url).then(() => {
            showToast("Link copied to clipboard!");
        });
    });

    // Sort timezones button
    sortBtn.addEventListener("click", sortTimezones);
}

// Show temporary Toast alerts
function showToast(msg) {
    let toast = document.querySelector(".toast-msg");
    if (!toast) {
        toast = document.createElement("div");
        toast.className = "toast-msg";
        document.body.appendChild(toast);
    }
    toast.innerHTML = `<i class="fa-solid fa-circle-check" style="color: var(--accent-teal)"></i> ${msg}`;
    toast.classList.add("show");
    setTimeout(() => {
        toast.classList.remove('show');
    }, 3000);
}

// Rename timezone
function renameTimezone(index) {
    const item = timezones[index];
    const newName = prompt("Enter a new display name for this location:", item.friendlyName);
    
    if (newName !== null && newName.trim() !== "") {
        item.friendlyName = newName.trim();
        updateUrl();
        render();
    }
}

// Set focus timezone
function setFocusTimezone(index) {
    const item = timezones[index];
    const identifier = item.searchTerm || item.tz;
    if (focusTz === identifier) {
        focusTz = null; // Toggle off
    } else {
        focusTz = identifier;
    }
    updateUrl();
    render();
}

// Sort timezones by raw UTC offset
function sortTimezones() {
    updateDOMWithTransition(() => {
        timezones.sort((a, b) => {
            const offsetA = getRawOffsetMinutes(selectedDate, a.tz);
            const offsetB = getRawOffsetMinutes(selectedDate, b.tz);
            return offsetA - offsetB;
        });
        
        updateUrl();
        render();
    });
}

let searchTimeout = null;

function handleSearchInput() {
    const val = tzSearch.value.trim();
    if (!val) {
        clearSearch.classList.add("hide");
        autocompleteList.classList.add("hide");
        return;
    }
    clearSearch.classList.remove("hide");

    // Debounce the search query to the backend
    if (searchTimeout) {
        clearTimeout(searchTimeout);
    }

    searchTimeout = setTimeout(() => {
        fetch(`/api/search?q=${encodeURIComponent(val)}`)
            .then(res => res.json())
            .then(matches => {
                // If input represents a valid timezone that isn't in database, offer it directly
                const normalizedVal = normalizeTzName(val);
                if (isValidTimeZone(normalizedVal) && !matches.some(m => m.zone.toLowerCase() === normalizedVal.toLowerCase())) {
                    matches.unshift({
                        name: normalizedVal.split('/').pop().replace(/_/g, ' '),
                        zone: normalizedVal,
                        desc: "Custom Timezone Region"
                    });
                }

                if (matches.length === 0) {
                    autocompleteList.innerHTML = `<div class="autocomplete-item"><span class="zone-title">No timezones found</span></div>`;
                } else {
                    autocompleteList.innerHTML = matches.map(item => {
                        const originalName = item.name;
                        const searchTermExact = val;

                        // Options set to deduplicate
                        const options = new Set();
                        options.add(originalName);

                        // User's exact search string (capitalized nicely)
                        const searchFormatted = searchTermExact.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
                        options.add(searchFormatted);

                        if (item.matchedAlt) {
                            options.add(item.matchedAlt);
                        }

                        const optionsHTML = Array.from(options).map(opt => 
                            `<button class="name-option" data-zone="${item.zone}" data-name="${opt}" data-searchterm="${originalName}">${opt}</button>`
                        ).join("");

                        return `
                            <div class="autocomplete-item">
                                <div style="display: flex; flex-direction: column; width: 100%;">
                                    <div style="display: flex; justify-content: space-between; align-items: center;">
                                        <div>
                                            <span class="zone-title">${item.name}</span>
                                            <span class="zone-sub">${item.desc}</span>
                                        </div>
                                    </div>
                                    <div class="item-options">
                                        <span class="options-label">Add as:</span>
                                        <div class="options-list">
                                            ${optionsHTML}
                                        </div>
                                    </div>
                                </div>
                            </div>
                        `;
                    }).join("");

                    // Attach click listeners to option buttons
                    const optionBtns = autocompleteList.querySelectorAll(".name-option");
                    optionBtns.forEach(btn => {
                        btn.addEventListener("click", (e) => {
                            e.stopPropagation();
                            const zone = btn.getAttribute("data-zone");
                            const name = btn.getAttribute("data-name");
                            const searchTerm = btn.getAttribute("data-searchterm");
                            if (zone && name) {
                                addTimezone(zone, name, searchTerm);
                            }
                        });
                    });
                }
                autocompleteList.classList.remove("hide");
            })
            .catch(err => {
                console.error("Error fetching search results:", err);
            });
    }, 150);
}

// Add timezone to list
function addTimezone(zone, friendlyName, searchTerm) {
    // Add zone and update layout using View Transitions if available
    updateDOMWithTransition(() => {
        timezones.push({ tz: zone, friendlyName: friendlyName, searchTerm: searchTerm || friendlyName });
        updateUrl();
        render();
    });

    tzSearch.value = "";
    clearSearch.classList.add("hide");
    autocompleteList.classList.add("hide");
}

// Remove timezone from list
function removeTimezone(index) {
    if (timezones.length <= 1) {
        showToast("Must keep at least one timezone!");
        return;
    }
    updateDOMWithTransition(() => {
        timezones.splice(index, 1);
        updateUrl();
        render();
    });
}

// Helper to trigger View Transitions when modifying list items
function updateDOMWithTransition(callback) {
    if (document.startViewTransition) {
        document.startViewTransition(callback);
    } else {
        callback();
    }
}

// HTML Renderer for the list rows
function render() {
    if (timezones.length === 0) {
        tzListContainer.innerHTML = `<p style="text-align: center; padding: 2rem; color: var(--text-muted)">Add a timezone to get started</p>`;
        return;
    }

    let focusZone = timezones[0];
    if (focusTz) {
        const found = timezones.find(t => (t.searchTerm === focusTz) || (t.tz === focusTz) || (t.friendlyName === focusTz));
        if (found) focusZone = found;
    }
    const firstTz = focusZone.tz;
    const dateStr = selectedDate.toISOString().split('T')[0];

    // Preserve scroll position for smooth centering
    const oldScrolls = [];
    if (window.innerWidth <= 1024) {
        tzListContainer.querySelectorAll(".row-right").forEach(r => {
            oldScrolls.push(r.scrollLeft);
        });
    }

    // Compute UTC time coordinates for 00:00 to 23:00 hours in the base (first) timezone
    const hourTimestamps = [];
    for (let h = 0; h < 24; h++) {
        hourTimestamps.push(getDateTimeInZone(selectedDate, h, firstTz));
    }

    // Render each timezone row
    tzListContainer.innerHTML = timezones.map((item, index) => {
        const tz = item.tz;
        const friendlyName = item.friendlyName;

        // Find timezone details for the currently SELECTED column hour
        const selectedTimeUTC = hourTimestamps[selectedHour];
        const selectedParts = getTzParts(selectedTimeUTC, tz);

        // Find timezone details for the ACTUAL current time
        const nowUTC = new Date();
        const currentParts = getTzParts(nowUTC, tz);

        // Format base current display time (left card text)
        const currentHourPad = String(currentParts.hour).padStart(2, '0');
        const currentMinPad = String(currentParts.minute).padStart(2, '0');

        let displayTime = "";
        if (is12hFormat) {
            let h12 = currentParts.hour % 12;
            if (h12 === 0) h12 = 12;
            const period = currentParts.hour >= 12 ? "PM" : "AM";
            displayTime = `${String(h12).padStart(2, '0')}:${currentMinPad} ${period}`;
        } else {
            displayTime = `${currentHourPad}:${currentMinPad}`;
        }

        // Format Date text (using selectedParts to shift date correctly when sliding)
        const months = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
        const dateText = `${months[selectedParts.month - 1]} ${selectedParts.day}`;

        // Get relative offset string from the focus timezone
        const relativeOffsetVal = getRelativeOffsetStr(selectedTimeUTC, tz, firstTz);

        let isFocus = false;
        if (focusTz) {
            isFocus = (item.searchTerm === focusTz || item.tz === focusTz || item.friendlyName === focusTz);
        } else {
            isFocus = (index === 0);
        }

        // Build HTML for timeline cells (24 hours)
        let hoursHTML = "";
        for (let col = 0; col < 24; col++) {
            const cellUTC = hourTimestamps[col];
            const cellParts = getTzParts(cellUTC, tz);

            const cellHour = cellParts.hour;
            const cellMin = cellParts.minute;

            let typeClass = "night";
            if (cellHour >= 9 && cellHour < 17) {
                typeClass = "work";
            } else if ((cellHour >= 17 && cellHour < 22) || (cellHour >= 6 && cellHour < 9)) {
                typeClass = "transition";
            }
            const activeClass = (col === selectedHour) ? "active-hour" : "";

            let boundaryClass = "";
            const baseYMD = `${currentParts.year}-${currentParts.month}-${currentParts.day}`;
            const cellYMD = `${cellParts.year}-${cellParts.month}-${cellParts.day}`;

            if (cellYMD !== baseYMD) {
                const cellT = Date.UTC(cellParts.year, cellParts.month - 1, cellParts.day);
                const baseT = Date.UTC(currentParts.year, currentParts.month - 1, currentParts.day);
                boundaryClass = cellT > baseT ? "date-boundary-next" : "date-boundary-prev";
            }

            let cellNum = cellHour;
            let cellPeriod = "";
            let cellMinText = "";
            if (is12hFormat) {
                cellNum = cellHour % 12;
                if (cellNum === 0) cellNum = 12;
                cellPeriod = cellHour >= 12 ? "pm" : "am";
                if (cellMin !== 0) {
                    cellMinText = String(cellMin).padStart(2, '0');
                }
            } else {
                cellNum = String(cellHour).padStart(2, '0');
                if (cellMin !== 0) {
                    cellMinText = String(cellMin).padStart(2, '0');
                }
            }

            hoursHTML += `
                <div class="hour-cell ${typeClass} ${activeClass} ${boundaryClass}" data-col="${col}">
                    <span class="hour-number">${cellNum}</span>
                    ${cellMinText ? `<span class="hour-minute">${cellMinText}</span>` : ""}
                    <span class="hour-period">${cellPeriod}</span>
                </div>
            `;
        }

        return `
            <div class="timezone-row" style="view-transition-name: tz-row-${index}" data-searchterm="${item.searchTerm || item.friendlyName || item.tz}">
                <div class="row-left">
                    <i class="fa-solid fa-grip-vertical drag-handle"></i>
                    <div class="row-meta-info" style="display: flex; gap: 0.5rem; align-items: center; flex: 1;">
                        <div class="zone-details">
                            <span class="current-time-text">${displayTime}</span>
                            <span class="zone-name" title="${tz}">${friendlyName}</span>
                            <span class="zone-offset">(${relativeOffsetVal})</span>
                        </div>
                        <div class="zone-date">${dateText}</div>
                    </div>
                    <button class="focus-btn ${isFocus ? 'active' : ''}" onclick="setFocusTimezone(${index})" title="${isFocus ? 'Currently focused' : 'Set as focus'}">
                        <i class="fa-solid fa-crosshairs"></i>
                    </button>
                    <button class="rename-btn" onclick="renameTimezone(${index})" title="Rename timezone">
                        <i class="fa-solid fa-pen"></i>
                    </button>
                    <button class="delete-btn" onclick="removeTimezone(${index})" title="Delete timezone">
                        <i class="fa-solid fa-trash-can"></i>
                    </button>
                </div>
                <div class="row-right">
                    ${hoursHTML}
                </div>
            </div>
        `;
    }).join("");

    setupDragAndDrop();

    const cells = tzListContainer.querySelectorAll(".hour-cell");
    cells.forEach(el => {
        el.addEventListener("click", () => {
            const col = parseInt(el.getAttribute("data-col"));
            selectedHour = col;
            render();
        });
    });

    // Restore old scroll positions immediately before the browser paints
    if (window.innerWidth <= 1024) {
        const newRows = tzListContainer.querySelectorAll(".row-right");
        newRows.forEach((r, i) => {
            if (i < oldScrolls.length) {
                r.scrollLeft = oldScrolls[i];
            } else if (oldScrolls.length > 0) {
                // If a new row was added, sync it to the first row's scroll
                r.scrollLeft = oldScrolls[0];
            }
        });
    }

    // Center active hour on mobile layout
    centerActiveHours();
}

// Center the active hour in the middle of the screen for small mobile layouts
function centerActiveHours() {
    // Only applies if the layout allows scrolling (e.g. mobile)
    if (window.innerWidth > 1024) return;
    
    // We defer centering slightly to ensure DOM is fully painted and dimensions are accurate
    requestAnimationFrame(() => {
        const rowRights = tzListContainer.querySelectorAll(".row-right");
        rowRights.forEach(container => {
            const activeCell = container.querySelector(".active-hour");
            if (activeCell) {
                const containerCenter = container.clientWidth / 2;
                const cellCenter = activeCell.offsetLeft + (activeCell.clientWidth / 2);
                container.scrollTo({
                    left: cellCenter - containerCenter,
                    behavior: 'smooth'
                });
            }
        });
    });
}

// Drag & Drop reorder implementation
function setupDragAndDrop() {
    const rows = tzListContainer.querySelectorAll(".timezone-row");

    rows.forEach((row, idx) => {
        const handle = row.querySelector('.drag-handle');
        if (handle) {
            handle.addEventListener('mousedown', () => {
                row.setAttribute('draggable', 'true');
            });
            handle.addEventListener('mouseup', () => {
                row.removeAttribute('draggable');
            });
        }

        row.addEventListener('dragstart', (e) => {
            dragSrcEl = row;
            row.classList.add('dragging');
            e.dataTransfer.effectAllowed = 'move';
            e.dataTransfer.setData('text/html', row.innerHTML);
        });

        row.addEventListener('dragend', () => {
            row.removeAttribute('draggable');
            row.classList.remove('dragging');
            rows.forEach(r => r.classList.remove('drag-over'));

            const newRows = Array.from(tzListContainer.querySelectorAll(".timezone-row"));
            const newZonesList = [];

            newRows.forEach(r => {
                const tz = r.querySelector(".zone-name").getAttribute("title");
                const friendlyName = r.querySelector(".zone-name").innerText;
                const searchTerm = r.getAttribute("data-searchterm") || tz;
                newZonesList.push({ tz, friendlyName, searchTerm });
            });

            timezones = newZonesList;
            updateUrl();
        });

        row.addEventListener('dragover', (e) => {
            e.preventDefault();
            return false;
        });

        row.addEventListener('dragenter', (e) => {
            if (row !== dragSrcEl) {
                row.classList.add('drag-over');
            }
        });

        row.addEventListener('dragleave', () => {
            row.classList.remove('drag-over');
        });

        row.addEventListener('drop', (e) => {
            e.stopPropagation();

            if (dragSrcEl !== row) {
                updateDOMWithTransition(() => {
                    const allRows = Array.from(tzListContainer.querySelectorAll(".timezone-row"));
                    const srcIndex = allRows.indexOf(dragSrcEl);
                    const destIndex = allRows.indexOf(row);

                    if (srcIndex < destIndex) {
                        tzListContainer.insertBefore(dragSrcEl, row.nextSibling);
                    } else {
                        tzListContainer.insertBefore(dragSrcEl, row);
                    }
                });
            }
            return false;
        });
    });
}

// Start application
window.onload = init;

// Periodically update the live clocks in the UI every 15 seconds without full rerenders
setInterval(() => {
    const rows = tzListContainer.querySelectorAll(".timezone-row");
    timezones.forEach((item, index) => {
        const row = rows[index];
        if (!row) return;
        const timeEl = row.querySelector(".current-time-text");

        const nowUTC = new Date();
        const currentParts = getTzParts(nowUTC, item.tz);
        const currentHourPad = String(currentParts.hour).padStart(2, '0');
        const currentMinPad = String(currentParts.minute).padStart(2, '0');

        let displayTime = "";
        if (is12hFormat) {
            let h12 = currentParts.hour % 12;
            if (h12 === 0) h12 = 12;
            const period = currentParts.hour >= 12 ? "PM" : "AM";
            displayTime = `${String(h12).padStart(2, '0')}:${currentMinPad} ${period}`;
        } else {
            displayTime = `${currentHourPad}:${currentMinPad}`;
        }

        if (timeEl && timeEl.textContent !== displayTime) {
            timeEl.textContent = displayTime;
        }
    });
}, 15000);
