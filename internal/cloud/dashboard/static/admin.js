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
