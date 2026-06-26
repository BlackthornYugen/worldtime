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

// Autocomplete database of common timezone abbreviations and cities
const SearchDatabase = [
    { name: "UTC", zone: "UTC", desc: "Coordinated Universal Time" },
    { name: "GMT", zone: "Etc/GMT", desc: "Greenwich Mean Time" },
    { name: "EST", zone: "America/New_York", desc: "Eastern Standard Time" },
    { name: "EDT", zone: "America/New_York", desc: "Eastern Daylight Time" },
    { name: "CST", zone: "America/Chicago", desc: "Central Standard Time" },
    { name: "CDT", zone: "America/Chicago", desc: "Central Daylight Time" },
    { name: "MST", zone: "America/Denver", desc: "Mountain Standard Time" },
    { name: "MDT", zone: "America/Denver", desc: "Mountain Daylight Time" },
    { name: "PST", zone: "America/Los_Angeles", desc: "Pacific Standard Time" },
    { name: "PDT", zone: "America/Los_Angeles", desc: "Pacific Daylight Time" },
    { name: "CEST", zone: "Europe/Paris", desc: "Central European Summer Time" },
    { name: "CET", zone: "Europe/Paris", desc: "Central European Time" },
    { name: "BST", zone: "Europe/London", desc: "British Summer Time" },
    { name: "JST", zone: "Asia/Tokyo", desc: "Japan Standard Time" },
    { name: "IST", zone: "Asia/Kolkata", desc: "India Standard Time" },
    { name: "AEST", zone: "Australia/Sydney", desc: "Australian Eastern Standard Time" },
    { name: "AEDT", zone: "Australia/Sydney", desc: "Australian Eastern Daylight Time" },
    { name: "New York", zone: "America/New_York", desc: "Eastern Time (US)" },
    { name: "Los Angeles", zone: "America/Los_Angeles", desc: "Pacific Time (US)" },
    { name: "Chicago", zone: "America/Chicago", desc: "Central Time (US)" },
    { name: "Denver", zone: "America/Denver", desc: "Mountain Time (US)" },
    { name: "Phoenix", zone: "America/Phoenix", desc: "Mountain Standard Time (No DST)" },
    { name: "Anchorage", zone: "America/Anchorage", desc: "Alaska Time" },
    { name: "Honolulu", zone: "Pacific/Honolulu", desc: "Hawaii Standard Time" },
    { name: "London", zone: "Europe/London", desc: "London, GMT/BST" },
    { name: "Paris", zone: "Europe/Paris", desc: "Paris, CET/CEST" },
    { name: "Berlin", zone: "Europe/Berlin", desc: "Berlin, CET/CEST" },
    { name: "Rome", zone: "Europe/Rome", desc: "Rome, CET/CEST" },
    { name: "Athens", zone: "Europe/Athens", desc: "Athens, EET/EEST" },
    { name: "Moscow", zone: "Europe/Moscow", desc: "Moscow Time" },
    { name: "Tokyo", zone: "Asia/Tokyo", desc: "Japan, Tokyo" },
    { name: "Seoul", zone: "Asia/Seoul", desc: "Korea, Seoul" },
    { name: "Singapore", zone: "Asia/Singapore", desc: "Singapore" },
    { name: "Hong Kong", zone: "Asia/Hong_Kong", desc: "Hong Kong" },
    { name: "Shanghai", zone: "Asia/Shanghai", desc: "China, Beijing" },
    { name: "Kolkata", zone: "Asia/Kolkata", desc: "India Standard Time" },
    { name: "Sydney", zone: "Australia/Sydney", desc: "Sydney, AEST/AEDT" },
    { name: "Melbourne", zone: "Australia/Melbourne", desc: "Melbourne, AEST/AEDT" },
    { name: "Perth", zone: "Australia/Perth", desc: "Perth, AWST" },
    { name: "Auckland", zone: "Pacific/Auckland", desc: "New Zealand, Auckland" },
    { name: "Dubai", zone: "Asia/Dubai", desc: "Gulf Standard Time" },
    { name: "Cairo", zone: "Africa/Cairo", desc: "Egypt, Cairo" },
    { name: "Johannesburg", zone: "Africa/Johannesburg", desc: "South Africa" },
    { name: "Sao Paulo", zone: "America/Sao_Paulo", desc: "Brazil, Sao Paulo" },
];

// App state
let timezones = []; // Array of objects: { tz: string, friendlyName: string }
let selectedHour = new Date().getHours(); // 0-23
let selectedDate = new Date(); // Date object
let is12hFormat = true;
let dragSrcEl = null;

// DOM Elements
const tzSearch = document.getElementById("tz-search");
const clearSearch = document.getElementById("clear-search");
const autocompleteList = document.getElementById("autocomplete-list");
const dateSelect = document.getElementById("date-select");
const format12hBtn = document.getElementById("format-12h");
const format24hBtn = document.getElementById("format-24h");
const shareBtn = document.getElementById("share-btn");
const tzListContainer = document.getElementById("tz-list");

// Init application
function init() {
    // Set date input to today
    const yyyy = selectedDate.getFullYear();
    const mm = String(selectedDate.getMonth() + 1).padStart(2, '0');
    const dd = String(selectedDate.getDate()).padStart(2, '0');
    dateSelect.value = `${yyyy}-${mm}-${dd}`;

    // Read URL query parameters first, then path segments as fallback
    const queryZones = parseQueryParams();
    if (queryZones.length > 0) {
        timezones = queryZones;
    } else {
        const pathZones = parseUrlPath();
        if (pathZones.length > 0) {
            timezones = pathZones;
            updateUrl();
        } else {
            // Default list: client local + UTC + other key zones
            const localZone = Intl.DateTimeFormat().resolvedOptions().timeZone;
            timezones = [
                { tz: localZone, friendlyName: "Local" },
                { tz: "UTC", friendlyName: "UTC" },
                { tz: "America/New_York", friendlyName: "New York" },
                { tz: "Europe/Paris", friendlyName: "Paris" },
                { tz: "Asia/Tokyo", friendlyName: "Tokyo" }
            ];
            updateUrl();
        }
    }

    // Set selected hour to current local time of the first zone
    if (timezones.length > 0) {
        const parts = getTzParts(new Date(), timezones[0].tz);
        selectedHour = parts.hour;
    }

    // Set up event listeners
    setupEventListeners();

    // Render list
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

// Parse query params (e.g. ?tz=America/New_York&friendlyName=Waterloo)
function parseQueryParams() {
    const params = new URLSearchParams(window.location.search);
    const tzs = params.getAll("tz");
    const friendlyNames = params.getAll("friendlyName");
    
    const list = [];
    for (let i = 0; i < tzs.length; i++) {
        const tz = tzs[i];
        if (isValidTimeZone(tz)) {
            const friendlyName = friendlyNames[i] || getFriendlyName(tz);
            list.push({ tz, friendlyName });
        }
    }
    return list;
}

// Parse timezone list from window pathname (fallback)
function parseUrlPath() {
    const path = window.location.pathname;
    const segments = path.split('/').map(s => s.trim()).filter(s => s !== '');
    if (segments.length === 0) return [];

    const resolved = [];
    for (let i = 0; i < segments.length; ) {
        // Try 3 segments (e.g., America/North_Dakota/New_Salem)
        if (i + 2 < segments.length) {
            const candidate = segments.slice(i, i + 3).join('/');
            const normalized = normalizeTzName(candidate);
            if (isValidTimeZone(normalized)) {
                resolved.push({ tz: normalized, friendlyName: getFriendlyName(normalized) });
                i += 3;
                continue;
            }
        }
        // Try 2 segments (e.g., America/New_York)
        if (i + 1 < segments.length) {
            const candidate = segments.slice(i, i + 2).join('/');
            const normalized = normalizeTzName(candidate);
            if (isValidTimeZone(normalized)) {
                resolved.push({ tz: normalized, friendlyName: getFriendlyName(normalized) });
                i += 2;
                continue;
            }
        }
        // Try 1 segment
        const candidate = segments[i];
        const lower = candidate.toLowerCase();
        if (AbbreviationMap[lower]) {
            const tz = AbbreviationMap[lower];
            resolved.push({ tz, friendlyName: candidate.toUpperCase() });
        } else {
            const normalized = normalizeTzName(candidate);
            if (isValidTimeZone(normalized)) {
                resolved.push({ tz: normalized, friendlyName: getFriendlyName(normalized) });
            }
        }
        i += 1;
    }
    return resolved;
}

// Update address bar query parameters
function updateUrl() {
    const params = new URLSearchParams();
    timezones.forEach(item => {
        params.append("tz", item.tz);
        params.append("friendlyName", item.friendlyName);
    });
    // Replace URL keeping query parameters
    const newUrl = window.location.pathname + '?' + params.toString();
    history.replaceState(null, '', newUrl);
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
        toast.classList.remove("show");
    }, 2500);
}

// Handle fuzzy suggestions as the user types
function handleSearchInput() {
    const val = tzSearch.value.trim().toLowerCase();
    if (!val) {
        clearSearch.classList.add("hide");
        autocompleteList.classList.add("hide");
        return;
    }
    clearSearch.classList.remove("hide");

    // Filter database
    let matches = SearchDatabase.filter(item => {
        return item.name.toLowerCase().includes(val) || 
               item.zone.toLowerCase().includes(val) || 
               item.desc.toLowerCase().includes(val);
    }).slice(0, 8); // limit results

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
        autocompleteList.innerHTML = matches.map(item => `
            <div class="autocomplete-item" data-zone="${item.zone}" data-name="${item.name}">
                <div>
                    <span class="zone-title">${item.name}</span>
                    <span class="zone-sub">${item.desc}</span>
                </div>
                <i class="fa-solid fa-plus" style="color: var(--accent-teal)"></i>
            </div>
        `).join("");

        // Attach click listeners to suggestions
        const items = autocompleteList.querySelectorAll(".autocomplete-item");
        items.forEach(el => {
            el.addEventListener("click", () => {
                const zone = el.getAttribute("data-zone");
                const name = el.getAttribute("data-name");
                if (zone && name) {
                    addTimezone(zone, name);
                }
            });
        });
    }
    autocompleteList.classList.remove("hide");
}

// Add timezone to list
function addTimezone(zone, friendlyName) {
    if (timezones.some(item => item.tz === zone)) {
        showToast("Timezone already added!");
        return;
    }
    
    // Add zone and update layout using View Transitions if available
    updateDOMWithTransition(() => {
        timezones.push({ tz: zone, friendlyName: friendlyName });
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

    const firstTz = timezones[0].tz;
    const dateStr = selectedDate.toISOString().split('T')[0];

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
        const currentParts = getTzParts(selectedTimeUTC, tz);
        
        // Format base current display time (left card text)
        const currentHourPad = String(currentParts.hour).padStart(2, '0');
        const currentMinPad = String(currentParts.minute).padStart(2, '0');
        
        let displayTime = "";
        let displayPeriod = "";
        if (is12hFormat) {
            let h12 = currentParts.hour % 12;
            if (h12 === 0) h12 = 12;
            const period = currentParts.hour >= 12 ? "PM" : "AM";
            displayTime = `${String(h12).padStart(2, '0')}:${currentMinPad}`;
            displayPeriod = ` ${period}`;
        } else {
            displayTime = `${currentHourPad}:${currentMinPad}`;
        }

        // Format Date text
        const months = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
        const targetDateObj = new Date(Date.UTC(currentParts.year, currentParts.month - 1, currentParts.day));
        const dateText = `${months[currentParts.month - 1]} ${currentParts.day}`;

        // Get offset string
        const offsetVal = getOffsetStr(selectedTimeUTC, tz);
        
        // Fetch abbreviation name using browser formatting
        const formatterAbbr = new Intl.DateTimeFormat('en-US', { timeZone: tz, timeZoneName: 'short' });
        const abbrParts = formatterAbbr.formatToParts(selectedTimeUTC);
        const abbrVal = abbrParts.find(p => p.type === 'timeZoneName')?.value || "";

        // Build HTML for timeline cells (24 hours)
        let hoursHTML = "";
        for (let col = 0; col < 24; col++) {
            const cellUTC = hourTimestamps[col];
            const cellParts = getTzParts(cellUTC, tz);
            
            const cellHour = cellParts.hour;
            const cellMin = cellParts.minute;
            
            const isNight = (cellHour < 6 || cellHour >= 18);
            const typeClass = isNight ? "night" : "day";
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
            if (is12hFormat) {
                cellNum = cellHour % 12;
                if (cellNum === 0) cellNum = 12;
                cellPeriod = cellHour >= 12 ? "pm" : "am";
                if (cellMin !== 0) {
                    cellNum = `${cellNum}:${String(cellMin).padStart(2, '0')}`;
                }
            } else {
                cellNum = String(cellHour).padStart(2, '0');
                if (cellMin !== 0) {
                    cellNum = `${cellNum}:${String(cellMin).padStart(2, '0')}`;
                }
            }

            hoursHTML += `
                <div class="hour-cell ${typeClass} ${activeClass} ${boundaryClass}" data-col="${col}">
                    <span class="hour-number">${cellNum}</span>
                    <span class="hour-period">${cellPeriod}</span>
                </div>
            `;
        }

        return `
            <div class="timezone-row" draggable="true" style="view-transition-name: tz-row-${index}">
                <div class="row-left">
                    <i class="fa-solid fa-grip-vertical drag-handle"></i>
                    <div class="row-meta-info">
                        <div class="zone-details">
                            <span class="zone-name" title="${tz}">${friendlyName}</span>
                            <span class="zone-abbrev">${abbrVal}</span>
                            <span class="zone-offset">${offsetVal}</span>
                        </div>
                        <div class="zone-date">${dateText}</div>
                    </div>
                    <div class="row-time-display">
                        <span class="current-time-text">${displayTime}</span>
                        <span style="font-size: 0.8rem; color: var(--text-muted); font-weight:600;">${displayPeriod}</span>
                    </div>
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
}

// Drag & Drop reorder implementation
function setupDragAndDrop() {
    const rows = tzListContainer.querySelectorAll(".timezone-row");
    
    rows.forEach((row, idx) => {
        row.addEventListener('dragstart', (e) => {
            const path = e.composedPath();
            const isHandle = path.some(el => el.classList && el.classList.contains('drag-handle'));
            if (!isHandle) {
                e.preventDefault();
                return;
            }

            dragSrcEl = row;
            row.classList.add('dragging');
            e.dataTransfer.effectAllowed = 'move';
            e.dataTransfer.setData('text/html', row.innerHTML);
        });

        row.addEventListener('dragend', () => {
            row.classList.remove('dragging');
            rows.forEach(r => r.classList.remove('drag-over'));
            
            const newRows = Array.from(tzListContainer.querySelectorAll(".timezone-row"));
            const newZonesList = [];
            
            newRows.forEach(r => {
                const tz = r.querySelector(".zone-name").getAttribute("title");
                const friendlyName = r.querySelector(".zone-name").innerText;
                newZonesList.push({ tz, friendlyName });
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
