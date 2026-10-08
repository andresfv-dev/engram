// Admin dashboard dialogs: buttons with data-open-dialog / data-close-dialog
// open and close the <dialog> whose id they name. Clicking the backdrop also
// closes the dialog; Escape is handled natively by <dialog>.
(function () {
	document.addEventListener("click", function (event) {
		var opener = event.target.closest("[data-open-dialog]");
		if (opener) {
			var toOpen = document.getElementById(opener.getAttribute("data-open-dialog"));
			if (toOpen && typeof toOpen.showModal === "function") {
				toOpen.showModal();
				var first = toOpen.querySelector("input, select, textarea");
				if (first) first.focus();
			}
			return;
		}
		var closer = event.target.closest("[data-close-dialog]");
		if (closer) {
			var toClose = document.getElementById(closer.getAttribute("data-close-dialog"));
			if (toClose) toClose.close();
			return;
		}
		if (event.target instanceof HTMLDialogElement && event.target.classList.contains("admin-dialog")) {
			var rect = event.target.getBoundingClientRect();
			var inside = event.clientX >= rect.left && event.clientX <= rect.right && event.clientY >= rect.top && event.clientY <= rect.bottom;
			if (!inside) event.target.close();
		}
	});
})();

// Success toast (admin-ui-redesign T7): the toast rendered by
// LayoutWithNotice auto-dismisses after ~4s, can be closed early, and its
// "notice" query param is stripped from the address bar via
// history.replaceState so a reload never re-shows the same toast.
(function () {
	function dismissToast(toast) {
		if (toast && toast.parentNode) toast.parentNode.removeChild(toast);
	}

	function initToast() {
		var toast = document.querySelector("[data-toast]");
		if (!toast) return;
		var closeBtn = toast.querySelector("[data-toast-close]");
		if (closeBtn) {
			closeBtn.addEventListener("click", function () {
				dismissToast(toast);
			});
		}
		setTimeout(function () {
			dismissToast(toast);
		}, 4000);
		if (window.history && typeof window.history.replaceState === "function") {
			try {
				var url = new URL(window.location.href);
				if (url.searchParams.has("notice")) {
					url.searchParams.delete("notice");
					var next = url.pathname + (url.search ? url.search : "") + url.hash;
					window.history.replaceState(window.history.state, "", next);
				}
			} catch (e) {
				// Unsupported URL API or blocked history access: leave the
				// query param in place rather than breaking the page.
			}
		}
	}

	if (document.readyState === "loading") {
		document.addEventListener("DOMContentLoaded", initToast);
	} else {
		initToast();
	}
})();

// Sidebar groups (sidebar-nav T2): the <details> groups work without JS. With
// JS, a collapsed or expanded choice is remembered per browser. The group that
// holds the current page is rendered open by the server and is never forced
// closed by a stored value. Storage may be unavailable (private mode, blocked
// site data), so every access is guarded and failure is silent.
(function () {
	var PREFIX = "engram.nav.";

	function readState(name) {
		try {
			return window.localStorage.getItem(PREFIX + name);
		} catch (e) {
			return null;
		}
	}

	function writeState(name, open) {
		try {
			window.localStorage.setItem(PREFIX + name, open ? "open" : "closed");
		} catch (e) {
			// Storage blocked or full: the group still toggles, it just is not remembered.
		}
	}

	function initNavGroups() {
		var groups = document.querySelectorAll("details[data-nav-group]");
		Array.prototype.forEach.call(groups, function (group) {
			var name = group.getAttribute("data-nav-group");
			var holdsCurrentPage = group.querySelector('[aria-current="page"]') !== null;
			var stored = readState(name);
			if (!holdsCurrentPage && stored === "open") group.open = true;
			if (!holdsCurrentPage && stored === "closed") group.open = false;
			group.addEventListener("toggle", function () {
				writeState(name, group.open);
			});
		});
	}

	if (document.readyState === "loading") {
		document.addEventListener("DOMContentLoaded", initNavGroups);
	} else {
		initNavGroups();
	}
})();

// Sidebar drawer (sidebar-nav T3): below 900px the sidebar is an off-canvas
// drawer opened by the menu button. The "drawer-enabled" class is only added
// here, so without JS the sidebar stays a normal stacked block and nothing is
// hidden. Escape, the backdrop and following a link close the drawer.
(function () {
	function initDrawer() {
		var body = document.body;
		var toggle = document.querySelector("[data-sidebar-toggle]");
		var sidebar = document.getElementById("shell-sidebar");
		var backdrop = document.querySelector(".sidebar-backdrop");
		if (!toggle || !sidebar) return;
		body.classList.add("drawer-enabled");

		function setOpen(open) {
			body.classList.toggle("drawer-open", open);
			toggle.setAttribute("aria-expanded", open ? "true" : "false");
			toggle.setAttribute("aria-label", open ? "Close navigation" : "Open navigation");
			if (backdrop) backdrop.hidden = !open;
			if (open) {
				var firstLink = sidebar.querySelector("a, summary, button");
				if (firstLink) firstLink.focus();
			} else if (document.activeElement && sidebar.contains(document.activeElement)) {
				toggle.focus();
			}
		}

		toggle.addEventListener("click", function () {
			setOpen(!body.classList.contains("drawer-open"));
		});
		document.addEventListener("click", function (event) {
			if (!body.classList.contains("drawer-open")) return;
			if (event.target.closest("[data-sidebar-close]") || event.target.closest("#shell-sidebar a")) {
				setOpen(false);
			}
		});
		document.addEventListener("keydown", function (event) {
			if (event.key === "Escape" && body.classList.contains("drawer-open")) setOpen(false);
		});
		window.addEventListener("resize", function () {
			if (window.innerWidth > 900 && body.classList.contains("drawer-open")) setOpen(false);
		});
	}

	if (document.readyState === "loading") {
		document.addEventListener("DOMContentLoaded", initDrawer);
	} else {
		initDrawer();
	}
})();
