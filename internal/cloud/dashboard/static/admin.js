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
